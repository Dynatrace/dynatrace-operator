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
	"sync"
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
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
	mount "k8s.io/mount-utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Server struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access

	publishers map[string]csivolumes.Publisher
	opts       dtcsi.CSIOptions
	path       metadata.PathResolver
}

var _ csi.IdentityServer = &Server{}
var _ csi.NodeServer = &Server{}

func NewServer(opts dtcsi.CSIOptions, db metadata.Access) *Server {
	return &Server{
		opts:    opts,
		fs:      afero.Afero{Fs: afero.NewOsFs()},
		mounter: mount.New(""),
		db:      db,
		path:    metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (svr *Server) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(svr)
}

func (svr *Server) Start(ctx context.Context) error {
	// start registration and csi server in parallel
	ctx, cancelFunc := context.WithCancel(ctx)

	var wg sync.WaitGroup

	numberOfGoRoutines := 2
	wg.Add(numberOfGoRoutines)

	go svr.startCSIDriver(ctx, cancelFunc, &wg)
	go startRegistrationServer(ctx, cancelFunc, &wg)

	wg.Wait()

	return nil
}

func (svr *Server) startCSIDriver(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	defer metadata.LogAccessOverview(svr.db)
	defer cancel()
	defer wg.Done()

	proto, addr, err := parseEndpoint(svr.opts.Endpoint)
	if err != nil {
		log.Error(err, "failed to parse endpoint", "endpoint", svr.opts.Endpoint)

		return
	}

	if proto == "unix" {
		if err := svr.fs.Remove(addr); err != nil && !os.IsNotExist(err) {
			log.Error(err, "failed to remove old endpoint", "address", addr)

			return
		}
	}

	svr.publishers = map[string]csivolumes.Publisher{
		appvolumes.Mode:  appvolumes.NewAppVolumePublisher(svr.fs, svr.mounter, svr.db, svr.path),
		hostvolumes.Mode: hostvolumes.NewHostVolumePublisher(svr.fs, svr.mounter, svr.db, svr.path),
	}

	log.Info("starting csi server", "protocol", proto, "address", addr)

	listener, err := net.Listen(proto, addr)
	if err != nil {
		log.Error(err, "failed to listen on address", "address", addr)

		return
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

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	go func() {
		log.Info("listening for connections on address", "address", listener.Addr())

		if err := server.Serve(listener); err != nil {
			log.Error(err, "failed to serve")
		}
	}()

	<-ctx.Done()
	server.GracefulStop()
}

func startRegistrationServer(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	defer cancel()
	defer wg.Done()

	socketPath := buildRegistrationSocketPath()
	if err := CleanupSocketFile(socketPath); err != nil {
		log.Error(err, "failed to clean up socket file")

		return
	}

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Error(err, "failed to listen on socket")

		return
	}

	server := grpc.NewServer()
	registrar := newRegistrationServer(dtcsi.DriverName, "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/csi.sock", []string{"v1.0.0"}) // TODO remove hardcoding
	registerapi.RegisterRegistrationServer(server, registrar)

	go func() {
		log.Info("starting registration server", "address", lis.Addr())

		if err := server.Serve(lis); err != nil {
			log.Error(err, "failed to serve")
		}
	}()

	<-ctx.Done()
	server.GracefulStop()
}

func buildRegistrationSocketPath() string {
	return fmt.Sprintf("/registration/%s-reg.sock", dtcsi.DriverName)
}

func (svr *Server) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (svr *Server) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (svr *Server) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func (svr *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeCfg, err := csivolumes.ParseNodePublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	if isMounted, err := svr.mounter.IsMountPoint(volumeCfg.TargetPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if isMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	publisher, ok := svr.publishers[volumeCfg.Mode]
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

func (svr *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeInfo, err := csivolumes.ParseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	for _, publisher := range svr.publishers {
		canUnpublish, err := publisher.CanUnpublishVolume(ctx, volumeInfo)
		if err != nil {
			log.Error(err, "couldn't determine if volume can be unpublished", "publisher", publisher)
		}

		if canUnpublish {
			response, err := publisher.UnpublishVolume(ctx, volumeInfo)
			if err != nil {
				return nil, err
			}

			return response, nil
		}
	}

	svr.unmountUnknownVolume(*volumeInfo)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (svr *Server) unmountUnknownVolume(volumeInfo csivolumes.VolumeInfo) {
	log.Info("VolumeID not present in the database", "volumeID", volumeInfo.VolumeID, "targetPath", volumeInfo.TargetPath)

	if err := svr.mounter.Unmount(volumeInfo.TargetPath); err != nil {
		log.Error(err, "Tried to unmount unknown volume", "volumeID", volumeInfo.VolumeID)
	}
}

func (svr *Server) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: svr.opts.NodeId}, nil
}

func (svr *Server) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{Capabilities: []*csi.NodeServiceCapability{}}, nil
}

func (svr *Server) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
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
