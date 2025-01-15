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
	"runtime"
	"strings"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/app"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Server struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

	fs      afero.Afero
	mounter mount.Interface

	publishers map[string]csivolumes.Publisher
	opts       dtcsi.CSIOptions
	path       metadata.PathResolver
}

var _ csi.IdentityServer = &Server{}
var _ csi.NodeServer = &Server{}

func NewServer(opts dtcsi.CSIOptions) *Server {
	return &Server{
		opts:    opts,
		fs:      afero.Afero{Fs: afero.NewOsFs()},
		mounter: mount.New(""),
		path:    metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (srv *Server) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(srv)
}

func (srv *Server) Start(ctx context.Context) error {
	proto, addr, err := parseEndpoint(srv.opts.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint '%s': %w", srv.opts.Endpoint, err)
	}

	if proto == "unix" {
		if err := srv.fs.Remove(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old endpoint on '%s': %w", addr, err)
		}
	}

	srv.publishers = map[string]csivolumes.Publisher{
		appvolumes.Mode:  appvolumes.NewPublisher(srv.fs, srv.mounter, srv.path),
		hostvolumes.Mode: hostvolumes.NewPublisher(srv.fs, srv.mounter, srv.path),
	}

	log.Info("starting listener", "protocol", proto, "address", addr)

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
				log.Info("stopping server")
				server.GracefulStop()
				log.Info("stopped server")

				done = true
			case <-ticker.C:
				var m runtime.MemStats

				runtime.ReadMemStats(&m)
				memoryUsageMetric.Set(float64(m.Alloc))
			}
		}
	}()

	csi.RegisterIdentityServer(server, srv)
	csi.RegisterNodeServer(server, srv)

	log.Info("listening for connections on address", "address", listener.Addr())

	err = server.Serve(listener)
	server.GracefulStop()

	return err
}

func (srv *Server) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (srv *Server) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (srv *Server) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func (srv *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeCfg, err := csivolumes.ParseNodePublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	if isMounted, err := srv.mounter.IsMountPoint(volumeCfg.TargetPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if isMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	publisher, ok := srv.publishers[volumeCfg.Mode]
	if !ok {
		return nil, status.Error(codes.Internal, "unknown csi mode provided, mode="+volumeCfg.Mode)
	}

	log.Info("publishing volume",
		"csiMode", volumeCfg.Mode,
		"target", volumeCfg.TargetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volumeCfg.VolumeID,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	return publisher.PublishVolume(ctx, volumeCfg)
}

func (srv *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeInfo, err := csivolumes.ParseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	srv.unmount(*volumeInfo)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (srv *Server) unmount(volumeInfo csivolumes.VolumeInfo) {
	// targetPath always needs to be unmounted
	if err := srv.mounter.Unmount(volumeInfo.TargetPath); err != nil {
		log.Error(err, "Unmount failed", "path", volumeInfo.TargetPath)
	}

	appMountDir := srv.path.AppMountForID(volumeInfo.VolumeID)

	mappedDir := srv.path.AppMountMappedDir(volumeInfo.VolumeID) // Unmount follows symlinks, so no need to check for them here

	_, err := srv.fs.Stat(mappedDir)
	if os.IsNotExist(err) { // case for timed out mounts
		_ = srv.fs.RemoveAll(appMountDir)

		return
	} else if err != nil {
		log.Error(err, "unexpected error when checking for app mount folder, trying to unmount just to be sure")
	}

	if err := srv.mounter.Unmount(mappedDir); err != nil {
		// Just try to unmount, nothing really can go wrong, just have to handle errors
		log.Error(err, "Unmount failed", "path", mappedDir)
	} else {
		// special handling is needed, because after upgrade/restart the mappedDir will be still busy
		needsCleanUp := []string{
			srv.path.AppMountVarDir(volumeInfo.VolumeID),
			srv.path.AppMountWorkDir(volumeInfo.VolumeID),
		}

		for _, path := range needsCleanUp {
			err := srv.fs.RemoveAll(path) // you see correctly, we don't keep the logs of the app mounts, will keep them when they will have a use
			if err != nil {
				log.Error(err, "failed to clean up unmounted volume dir", "path", path)
			}
		}

		_ = srv.fs.RemoveAll(appMountDir) // try to cleanup fully, but lets not spam the logs with errors
	}

	podInfoSymlinkPath := srv.findPodInfoSymlink(volumeInfo) // cleaning up the pod-info symlink here is far more efficient instead of having to walk the whole fs during cleanup
	if podInfoSymlinkPath != "" {
		_ = srv.fs.Remove(podInfoSymlinkPath)
	}
}

func (srv *Server) findPodInfoSymlink(volumeInfo csivolumes.VolumeInfo) string {
	podInfoPath := srv.path.OverlayVarPodInfo(volumeInfo.VolumeID)

	podInfoBytes, err := srv.fs.ReadFile(srv.path.OverlayVarPodInfo(volumeInfo.VolumeID))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}

		log.Error(err, "failed to read pod-info file", "path", podInfoPath)
	}

	return string(podInfoBytes)
}

func (srv *Server) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (srv *Server) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (srv *Server) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: srv.opts.NodeId}, nil
}

func (srv *Server) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{Capabilities: []*csi.NodeServiceCapability{}}, nil
}

func (srv *Server) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (srv *Server) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func logGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == "/csi.v1.Identity/Probe" || info.FullMethod == "/csi.v1.Node/NodeGetCapabilities" {
			return handler(ctx, req)
		}

		methodName := ""

		if info.FullMethod == "/csi.v1.Node/NodePublishVolume" {
			req := req.(*csi.NodePublishVolumeRequest)
			methodName = "NodePublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.GetVolumeId())
		} else if info.FullMethod == "/csi.v1.Node/NodeUnpublishVolume" {
			req := req.(*csi.NodeUnpublishVolumeRequest)
			methodName = "NodeUnpublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.GetVolumeId())
		}

		resp, err := handler(ctx, req)
		if err != nil {
			log.Error(err, "GRPC call failed", "method", methodName)
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
