/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package csidriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	podNamespaceContextKey   = "csi.storage.k8s.io/pod.namespace"
	versionOffsetInUsagePath = -2
)

var (
	memoryUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "memory_usage",
		Help:      "Memory usage of the csi driver in bytes",
	})
	agentsVersionsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dynatrace",
		Subsystem: "csi_driver",
		Name:      "agent_versions",
		Help:      "Number of an agent version currently mounted by the CI driver",
	}, []string{"version"})
	memoryMetricTick = 5000 * time.Millisecond
)

func init() {
	metrics.Registry.MustRegister(memoryUsageMetric)
	metrics.Registry.MustRegister(agentsVersionsMetric)
}

var log = logger.NewDTLogger().WithName("server")

type CSIDriverServer struct {
	client  client.Client
	log     logr.Logger
	opts    dtcsi.CSIOptions
	fs      afero.Afero
	mounter mount.Interface
}

type volumeMetadata struct {
	UsageFilePath string
	OverlayFSPath string
}

var _ manager.Runnable = &CSIDriverServer{}
var _ csi.IdentityServer = &CSIDriverServer{}
var _ csi.NodeServer = &CSIDriverServer{}

func NewServer(client client.Client, opts dtcsi.CSIOptions) *CSIDriverServer {
	return &CSIDriverServer{
		client:  client,
		log:     log,
		opts:    opts,
		fs:      afero.Afero{Fs: afero.NewOsFs()},
		mounter: mount.New(""),
	}
}

func (svr *CSIDriverServer) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(svr)
}

func (svr *CSIDriverServer) Start(ctx context.Context) error {
	proto, addr, err := parseEndpoint(svr.opts.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint '%s': %w", svr.opts.Endpoint, err)
	}

	if proto == "unix" {
		if err := svr.fs.Remove(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old endpoint on '%s': %w", addr, err)
		}
	}

	svr.log.Info("Starting listener", "protocol", proto, "address", addr)

	listener, err := net.Listen(proto, addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(logGRPC(log)))
	go func() {
		ticker := time.NewTicker(memoryMetricTick)
		done := false
		for !done {
			select {
			case <-ctx.Done():
				svr.log.Info("Stopping server")
				server.GracefulStop()
				svr.log.Info("Stopped server")
				done = true
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				memoryUsageMetric.Set(float64(m.Alloc))
			}
		}
	}()

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	svr.log.Info("Listening for connections on address", "address", listener.Addr())

	_ = server.Serve(listener)

	return nil
}

func (svr *CSIDriverServer) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (svr *CSIDriverServer) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (svr *CSIDriverServer) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func isMounted(mounter mount.Interface, targetPath string) (bool, error) {
	isNotMounted, err := mount.IsNotMountPoint(mounter, targetPath)
	if os.IsNotExist(err) {
		isNotMounted = true
	} else if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}
	return !isNotMounted, nil
}

func (svr *CSIDriverServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeCfg, err := parsePublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	if isMounted, err := isMounted(svr.mounter, volumeCfg.targetPath); err != nil {
		return nil, err
	} else if isMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	svr.log.Info("Publishing volume",
		"target", volumeCfg.targetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volumeCfg.volumeId,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	bindCfg, err := newBindConfig(ctx, svr, volumeCfg, svr.fs)
	if err != nil {
		return nil, err
	}

	if err := svr.mountOneAgent(bindCfg, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := svr.storeVolumeMetadata(bindCfg, volumeCfg.volumeId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to store volume metadata: %s", err))
	}
	agentsVersionsMetric.WithLabelValues(bindCfg.version).Inc()
	return &csi.NodePublishVolumeResponse{}, nil
}

func (svr *CSIDriverServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID, targetPath, err := parseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	volumeMetadataPath := filepath.Join(svr.opts.RootDir, dtcsi.GarbageCollectionPath, volumeID)

	var metadata volumeMetadata
	if err = svr.loadVolumeMetadata(volumeMetadataPath, &metadata); err != nil {
		svr.log.Info("failed to load volume metadata", "error", err.Error())
	}

	if err = svr.umountOneAgent(targetPath, &metadata); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount oneagent volume: %s", err.Error()))
	}

	if strings.HasPrefix(metadata.UsageFilePath, svr.opts.RootDir) {
		if err = svr.fs.Remove(metadata.UsageFilePath); err != nil && !os.IsNotExist(err) {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove version reference for pod: %s", err))
		}
	}

	if err = svr.fs.Remove(volumeMetadataPath); err != nil && !os.IsNotExist(err) {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove volume to pod reference: %s", err))
	}

	if err = svr.fs.RemoveAll(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	svr.log.Info("volume has been unpublished", "targetPath", targetPath)

	svr.fireVolumeUnpublishedMetric(metadata.UsageFilePath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (svr *CSIDriverServer) fireVolumeUnpublishedMetric(usageFilePath string) {
	if len(usageFilePath) > 0 {
		tmp := strings.Split(usageFilePath, string(os.PathSeparator))
		version := tmp[len(tmp)+versionOffsetInUsagePath]
		agentsVersionsMetric.WithLabelValues(version).Dec()
		var m = &dto.Metric{}
		if err := agentsVersionsMetric.WithLabelValues(version).Write(m); err != nil {
			svr.log.Error(err, "Failed to get the value of agent version metric")
		}
		if m.Gauge.GetValue() <= float64(0) {
			agentsVersionsMetric.DeleteLabelValues(version)
		}
	}
}

func (svr *CSIDriverServer) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *CSIDriverServer) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *CSIDriverServer) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: svr.opts.NodeID}, nil
}

func (svr *CSIDriverServer) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{Capabilities: []*csi.NodeServiceCapability{}}, nil
}

func (svr *CSIDriverServer) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *CSIDriverServer) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *CSIDriverServer) mountOneAgent(bindCfg *bindConfig, volumeCfg *volumeConfig) error {
	agentDirectoryForPod := filepath.Join(bindCfg.envDir, "run", volumeCfg.volumeId)

	mappedDir := filepath.Join(agentDirectoryForPod, "mapped")
	_ = svr.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir := filepath.Join(agentDirectoryForPod, "var")
	_ = svr.fs.MkdirAll(upperDir, os.ModePerm)

	workDir := filepath.Join(agentDirectoryForPod, "work")
	_ = svr.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + bindCfg.agentDir,
		"upperdir=" + upperDir,
		"workdir=" + workDir,
	}

	if err := svr.fs.MkdirAll(volumeCfg.targetPath, os.ModePerm); err != nil {
		return err
	}

	if err := svr.mounter.Mount("overlay", mappedDir, "overlay", overlayOptions); err != nil {
		return err
	}
	if err := svr.mounter.Mount(mappedDir, volumeCfg.targetPath, "", []string{"bind"}); err != nil {
		_ = svr.mounter.Unmount(mappedDir)
		return err
	}

	return nil
}

func (svr *CSIDriverServer) umountOneAgent(targetPath string, metadata *volumeMetadata) error {
	if err := svr.mounter.Unmount(targetPath); err != nil {
		svr.log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(metadata.OverlayFSPath) {
		agentDirectoryForPod := filepath.Join(metadata.OverlayFSPath, "mapped")
		if err := svr.mounter.Unmount(agentDirectoryForPod); err != nil {
			svr.log.Error(err, "Unmount failed", "path", agentDirectoryForPod)
		}
	}

	return nil
}

func (svr *CSIDriverServer) storeVolumeMetadata(bindCfg *bindConfig, volumeID string) error {
	podToVersionReference := filepath.Join(bindCfg.volumeToVersionReferenceDir, volumeID)
	if err := svr.fs.WriteFile(podToVersionReference, nil, 0640); err != nil {
		return err
	}

	metadata := volumeMetadata{
		UsageFilePath: podToVersionReference,
		OverlayFSPath: filepath.Join(bindCfg.envDir, "run", volumeID),
	}

	volumeMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	metadataFile := filepath.Join(svr.opts.RootDir, dtcsi.GarbageCollectionPath, volumeID)
	if err = svr.fs.WriteFile(metadataFile, volumeMetadata, 0640); err != nil {
		return err
	}

	return nil
}

func (svr *CSIDriverServer) loadVolumeMetadata(metadataPath string, metadata *volumeMetadata) error {
	data, err := svr.fs.ReadFile(metadataPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &metadata)

	return err
}

func logGRPC(log logr.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/csi.v1.Identity/Probe" {
			return handler(ctx, req)
		}

		log.Info("GRPC call", "method", info.FullMethod, "request", req)

		resp, err := handler(ctx, req)
		if err != nil {
			log.Error(err, "GRPC call failed")
		} else {
			log.Info("GRPC call successful", "response", resp)
		}
		return resp, err
	}
}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func parseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (string, string, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return "", "", status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return "", "", status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return volumeID, targetPath, nil
}
