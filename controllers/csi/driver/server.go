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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/storage"
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
		Help:      "Number of an agent version currently mounted by the CSI driver",
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
	db      storage.Access
	fph     storage.FilePathHandler
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
		db:      storage.NewAccess(),
		fph:     storage.FilePathHandler{RootDir: opts.RootDir},
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

	bindCfg, err := newBindConfig(ctx, svr, volumeCfg, svr.db)
	if err != nil {
		return nil, err
	}

	if err := svr.mountOneAgent(bindCfg, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := svr.storeVolumeInfo(bindCfg, volumeCfg); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to store volume info: %s", err))
	}
	agentsVersionsMetric.WithLabelValues(bindCfg.version).Inc()
	return &csi.NodePublishVolumeResponse{}, nil
}

func (svr *CSIDriverServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID, targetPath, err := parseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	volume, err := svr.loadVolumeInfo(volumeID)
	if err != nil {
		svr.log.Info("failed to load volume info", "error", err.Error())
	}

	overlayFSPath := svr.fph.AgentRunDirForVolume(volume.TenantUUID, volumeID)

	if err = svr.umountOneAgent(targetPath, overlayFSPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount oneagent volume: %s", err.Error()))
	}

	if err = svr.db.DeleteVolumeInfo(volume.ID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	svr.log.Info("deleted volume info", "ID", volume.ID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)

	if err = svr.fs.RemoveAll(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	svr.log.Info("volume has been unpublished", "targetPath", targetPath)

	fireVolumeUnpublishedMetric(*volume)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func fireVolumeUnpublishedMetric(volume storage.Volume) {
	if len(volume.Version) > 0 {
		agentsVersionsMetric.WithLabelValues(volume.Version).Dec()
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
	mappedDir := svr.fph.OverlayMappedDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir := svr.fph.OverlayVarDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(upperDir, os.ModePerm)

	workDir := svr.fph.OverlayWorkDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + svr.fph.AgentBinaryDirForVersion(bindCfg.tenantUUID, bindCfg.version),
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

func (svr *CSIDriverServer) umountOneAgent(targetPath string, overlayFSPath string) error {
	if err := svr.mounter.Unmount(targetPath); err != nil {
		svr.log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(overlayFSPath) {
		agentDirectoryForPod := filepath.Join(overlayFSPath, dtcsi.OverlayMappedDirPath)
		if err := svr.mounter.Unmount(agentDirectoryForPod); err != nil {
			svr.log.Error(err, "Unmount failed", "path", agentDirectoryForPod)
		}
	}

	return nil
}

func (svr *CSIDriverServer) storeVolumeInfo(bindCfg *bindConfig, volumeCfg *volumeConfig) error {
	volume := storage.Volume{
		ID:         volumeCfg.volumeId,
		PodName:    volumeCfg.podName,
		Version:    bindCfg.version,
		TenantUUID: bindCfg.tenantUUID,
	}
	svr.log.Info("inserting volume info", "ID", volume.ID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)
	return svr.db.InsertVolumeInfo(&volume)
}

func (svr *CSIDriverServer) loadVolumeInfo(volumeID string) (*storage.Volume, error) {
	volume, err := svr.db.GetVolumeInfo(volumeID)
	if err != nil {
		return nil, err
	}
	if volume == nil {
		return &storage.Volume{}, nil
	}
	svr.log.Info("loaded volume info", "ID", volume.ID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)
	return volume, nil
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
