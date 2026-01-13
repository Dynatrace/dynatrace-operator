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

package csiserver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/host"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

const DefaultMaxGrpcRequests = 20

var counter atomic.Int32

type Server struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

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
		mounter: mount.New(""),
		path:    metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (srv *Server) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(srv)
}

func (srv *Server) Start(ctx context.Context) error {
	endpoint, err := url.Parse(srv.opts.Endpoint)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("failed to parse endpoint '%s'", srv.opts.Endpoint))
	}

	addr := endpoint.Host + endpoint.Path

	if endpoint.Scheme == "unix" {
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return errors.WithMessage(err, fmt.Sprintf("failed to remove old endpoint on '%s'", addr))
		}
	}

	srv.publishers = map[string]csivolumes.Publisher{
		appvolumes.Mode:  appvolumes.NewPublisher(srv.mounter, srv.path),
		hostvolumes.Mode: hostvolumes.NewPublisher(srv.mounter, srv.path),
	}

	log.Info("starting listener", "scheme", endpoint.Scheme, "address", addr)

	listener, err := (&net.ListenConfig{}).Listen(ctx, endpoint.Scheme, addr)
	if err != nil {
		return errors.WithMessage(err, "failed to start server")
	}

	maxGrpcRequests, err := strconv.ParseInt(os.Getenv("GRPC_MAX_REQUESTS_LIMIT"), 10, 32)
	if err != nil {
		maxGrpcRequests = DefaultMaxGrpcRequests
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(grpcLimiter(int32(maxGrpcRequests))))

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
	log.Debug("received GetPluginInfo")

	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (srv *Server) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	log.Debug("received Probe")

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

	return publisher.PublishVolume(ctx, &volumeCfg)
}

func (srv *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeInfo, err := csivolumes.ParseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	srv.unmount(volumeInfo)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (srv *Server) unmount(volumeInfo csivolumes.VolumeInfo) {
	_ = srv.unmountQuietly(volumeInfo.TargetPath)

	_ = srv.unmountQuietly(srv.path.AppMountMappedDir(volumeInfo.VolumeID))

	appMountDir := srv.path.AppMountForID(volumeInfo.VolumeID)
	needsCleanUp := []string{
		srv.path.AppMountVarDir(volumeInfo.VolumeID),
		srv.path.AppMountWorkDir(volumeInfo.VolumeID),
	}

	if err := srv.unmountQuietly(appMountDir); err == nil {
		podInfoSymlinkPath := srv.findPodInfoSymlink(volumeInfo)

		for _, path := range needsCleanUp {
			if podInfoSymlinkPath != "" {
				_ = os.Remove(podInfoSymlinkPath)
				podInfoSymlinkDir := filepath.Dir(podInfoSymlinkPath)

				if entries, _ := os.ReadDir(podInfoSymlinkDir); len(entries) == 0 {
					_ = os.Remove(podInfoSymlinkDir)
				}
			}

			if err := os.RemoveAll(path); err != nil {
				log.Error(err, "failed to clean up unmounted volume dir", "path", path)
			}
		}
	}

	_ = os.RemoveAll(appMountDir)
}

func (srv *Server) unmountQuietly(path string) error {
	if path == "" {
		return nil
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Debug("path already removed", "path", path)

			return nil
		}
	}

	err := srv.mounter.Unmount(path)
	if err == nil {
		return nil
	}

	if isOtherError(err) {
		log.Debug("already unmounted", "path", path, "err", err)

		return nil
	}

	// at this point we have an actual error we do not expect
	log.Error(err, "unmount failed", "path", path)

	return err
}

func isOtherError(err error) bool {
	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not mounted") ||
		strings.Contains(msg, "not a mount point") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "invalid argument")
}

func (srv *Server) findPodInfoSymlink(volumeInfo csivolumes.VolumeInfo) string {
	podInfoPath := srv.path.OverlayVarPodInfo(volumeInfo.VolumeID)

	podInfoBytes, err := os.ReadFile(srv.path.OverlayVarPodInfo(volumeInfo.VolumeID))
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
	return &csi.NodeGetInfoResponse{NodeId: srv.opts.NodeID}, nil
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

func grpcLimiter(maxGrpcRequests int32) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var methodName string

		var logValues []any

		switch info.FullMethod {
		case "/csi.v1.Node/NodePublishVolume":
			req := req.(*csi.NodePublishVolumeRequest)
			methodName = "NodePublishVolume"
			vc, _ := csivolumes.ParseNodePublishVolumeRequest(req)
			logValues = []any{
				"method", methodName,
				"volume-id", vc.VolumeID,
				"pod", vc.PodName,
				"namespace", vc.PodNamespace,
				"dynakube", vc.DynakubeName,
				"mode", vc.Mode,
			}
			log.Info("GRPC call", logValues...)
		case "/csi.v1.Node/NodeUnpublishVolume":
			req := req.(*csi.NodeUnpublishVolumeRequest)
			methodName = "NodeUnpublishVolume"
			vi, _ := csivolumes.ParseNodeUnpublishVolumeRequest(req)
			logValues = []any{ // this is all we get
				"method", methodName,
				"volume-id", vi.VolumeID,
				"target-path", vi.TargetPath,
			}
			log.Info("GRPC call", logValues...)
		default:
			log.Debug("GRPC call", "full_method", info.FullMethod)

			resp, err := handler(ctx, req)
			if err != nil {
				log.Info("GRPC failed", "full_method", info.FullMethod, "err", err.Error())
			}

			return resp, err
		}

		counter.Add(1)
		defer counter.Add(-1)

		if counter.Load() > maxGrpcRequests {
			msg := fmt.Sprintf("rate limit exceeded, current value %d more than max %d", counter.Load(), DefaultMaxGrpcRequests)

			log.Info(msg, logValues...)

			return nil, status.Error(codes.ResourceExhausted, msg)
		}

		resp, err := handler(ctx, req)
		if err != nil {
			logValues = append(logValues, "error", err.Error())

			log.Info("GRPC call failed", logValues...)
		}

		return resp, err
	}
}
