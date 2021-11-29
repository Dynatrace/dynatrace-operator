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
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type CSIDriverServer struct {
	client  client.Client
	opts    dtcsi.CSIOptions
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver
}

var _ manager.Runnable = &CSIDriverServer{}
var _ csi.IdentityServer = &CSIDriverServer{}
var _ csi.NodeServer = &CSIDriverServer{}

func NewServer(client client.Client, opts dtcsi.CSIOptions, db metadata.Access) *CSIDriverServer {
	return &CSIDriverServer{
		client:  client,
		opts:    opts,
		fs:      afero.Afero{Fs: afero.NewOsFs()},
		mounter: mount.New(""),
		db:      db,
		path:    metadata.PathResolver{RootDir: opts.RootDir},
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

	log.Info("Starting listener", "protocol", proto, "address", addr)

	listener, err := net.Listen(proto, addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(logGRPC()))
	go func() {
		ticker := time.NewTicker(memoryMetricTick)
		done := false
		for !done {
			select {
			case <-ctx.Done():
				log.Info("Stopping server")
				server.GracefulStop()
				log.Info("Stopped server")
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

	log.Info("Listening for connections on address", "address", listener.Addr())

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

	log.Info("Publishing volume",
		"target", volumeCfg.targetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volumeCfg.volumeId,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	bindCfg, err := newBindConfig(ctx, svr, volumeCfg)
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
		log.Info("failed to load volume info", "error", err.Error())
	}

	overlayFSPath := svr.path.AgentRunDirForVolume(volume.TenantUUID, volumeID)

	if err = svr.umountOneAgent(targetPath, overlayFSPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount oneagent volume: %s", err.Error()))
	}

	if err = svr.db.DeleteVolume(volume.VolumeID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Info("deleted volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)

	if err = svr.fs.RemoveAll(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info("volume has been unpublished", "targetPath", targetPath)

	svr.fireVolumeUnpublishedMetric(*volume)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (svr *CSIDriverServer) fireVolumeUnpublishedMetric(volume metadata.Volume) {
	if len(volume.Version) > 0 {
		agentsVersionsMetric.WithLabelValues(volume.Version).Dec()
		var m = &dto.Metric{}
		if err := agentsVersionsMetric.WithLabelValues(volume.Version).Write(m); err != nil {
			log.Error(err, "failed to get the value of agent version metric")
		}
		if m.Gauge.GetValue() <= float64(0) {
			agentsVersionsMetric.DeleteLabelValues(volume.Version)
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
	mappedDir := svr.path.OverlayMappedDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir := svr.path.OverlayVarDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(upperDir, os.ModePerm)

	workDir := svr.path.OverlayWorkDir(bindCfg.tenantUUID, volumeCfg.volumeId)
	_ = svr.fs.MkdirAll(workDir, os.ModePerm)

	overlayOptions := []string{
		"lowerdir=" + svr.path.AgentBinaryDirForVersion(bindCfg.tenantUUID, bindCfg.version),
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
		log.Error(err, "Unmount failed", "path", targetPath)
	}

	if filepath.IsAbs(overlayFSPath) {
		agentDirectoryForPod := filepath.Join(overlayFSPath, dtcsi.OverlayMappedDirPath)
		if err := svr.mounter.Unmount(agentDirectoryForPod); err != nil {
			log.Error(err, "Unmount failed", "path", agentDirectoryForPod)
		}
	}

	return nil
}

func (svr *CSIDriverServer) storeVolumeInfo(bindCfg *bindConfig, volumeCfg *volumeConfig) error {
	volume := metadata.NewVolume(volumeCfg.volumeId, volumeCfg.podName, bindCfg.version, bindCfg.tenantUUID)
	log.Info("inserting volume info", "ID", volume.VolumeID, "PodUID", volume.PodName, "Version", volume.Version, "TenantUUID", volume.TenantUUID)
	return svr.db.InsertVolume(volume)
}

func (svr *CSIDriverServer) loadVolumeInfo(volumeID string) (*metadata.Volume, error) {
	volume, err := svr.db.GetVolume(volumeID)
	if err != nil {
		return nil, err
	}
	if volume == nil {
		return &metadata.Volume{}, nil
	}
	log.Info("loaded volume info", "id", volume.VolumeID, "pod name", volume.PodName, "version", volume.Version, "dynakube", volume.TenantUUID)
	return volume, nil
}

func logGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/csi.v1.Identity/Probe" || info.FullMethod == "/csi.v1.Node/NodeGetCapabilities" {
			return handler(ctx, req)
		}
		methodName := ""
		if info.FullMethod == "/csi.v1.Node/NodePublishVolume" {
			req := req.(*csi.NodePublishVolumeRequest)
			methodName = "NodePublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.VolumeId)
		} else if info.FullMethod == "/csi.v1.Node/NodeUnpublishVolume" {
			req := req.(*csi.NodeUnpublishVolumeRequest)
			methodName = "NodeUnpublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.VolumeId)
		}
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error(err, fmt.Sprintf("%s GRPC call failed", methodName))
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
