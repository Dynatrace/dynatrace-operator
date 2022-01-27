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

	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CSIDriverServer struct {
	client  client.Client
	opts    dtcsi.CSIOptions
	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access
	path    metadata.PathResolver

	publishers map[string]csivolumes.Publisher
}

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

	svr.publishers = map[string]csivolumes.Publisher{
		appvolumes.Mode: appvolumes.NewPublisher(svr.client, svr.fs, svr.mounter, svr.db, svr.path),
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

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	log.Info("listening for connections on address", "address", listener.Addr())

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

func (svr *CSIDriverServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeCfg, err := csivolumes.ParseNodePublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}

	if isMounted, err := isMounted(svr.mounter, volumeCfg.TargetPath); err != nil {
		return nil, err
	} else if isMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	log.Info("publishing app volume",
		"target", volumeCfg.TargetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volumeCfg.VolumeId,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	response, err := svr.publishers[appvolumes.Mode].PublishVolume(ctx, volumeCfg)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (svr *CSIDriverServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeInfo, err := csivolumes.ParseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, err
	}
	response, err := svr.publishers[appvolumes.Mode].UnpublishVolume(ctx, volumeInfo)
	if err != nil {
		return nil, err
	}
	return response, nil
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

func isMounted(mounter mount.Interface, targetPath string) (bool, error) {
	isNotMounted, err := mount.IsNotMountPoint(mounter, targetPath)
	if os.IsNotExist(err) {
		isNotMounted = true
	} else if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}
	return !isNotMounted, nil
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
