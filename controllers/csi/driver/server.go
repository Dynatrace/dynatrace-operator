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
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/mount"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	podNamespaceContextKey = "csi.storage.k8s.io/pod.namespace"
	podUIDContextKey       = "csi.storage.k8s.io/pod.uid"
)

var log = logger.NewDTLogger().WithName("server")

type CSIDriverServer struct {
	client client.Client
	log    logr.Logger
	opts   dtcsi.CSIOptions
}

var _ manager.Runnable = &CSIDriverServer{}
var _ csi.IdentityServer = &CSIDriverServer{}
var _ csi.NodeServer = &CSIDriverServer{}

func NewServer(mgr ctrl.Manager, opts dtcsi.CSIOptions) *CSIDriverServer {
	return &CSIDriverServer{
		client: mgr.GetClient(),
		log:    log,
		opts:   opts,
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
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
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
		<-ctx.Done()
		svr.log.Info("Stopping server")
		server.GracefulStop()
		svr.log.Info("Stopped server")
	}()

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	svr.log.Info("Listening for connections on address", "address", listener.Addr())

	server.Serve(listener)

	return nil
}

// csi.IdentityServer implementation

func (svr *CSIDriverServer) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (svr *CSIDriverServer) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (svr *CSIDriverServer) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

// csi.NodeServer implementation

func (svr *CSIDriverServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}

	volID := req.GetVolumeId()
	if volID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if req.GetVolumeCapability().GetBlock() != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot have block access type")
	}

	if req.GetVolumeCapability().GetMount() == nil {
		return nil, status.Error(codes.InvalidArgument, "expecting to have mount access type")
	}

	volCtx := req.GetVolumeContext()
	if volCtx == nil {
		return nil, status.Error(codes.InvalidArgument, "Publish context missing in request")
	}

	nsName := volCtx[podNamespaceContextKey]
	if nsName == "" {
		return nil, status.Error(codes.InvalidArgument, "No namespace included with request")
	}

	podUID := volCtx[podUIDContextKey]
	if podUID == "" {
		return nil, status.Error(codes.InvalidArgument, "No Pod UID included with request")
	}

	flavor := volCtx["flavor"]
	if flavor == "" {
		flavor = dtclient.FlavorDefault
	}
	if flavor != dtclient.FlavorDefault && flavor != dtclient.FlavorMUSL {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid flavor in request: %s", flavor))
	}

	notMnt, err := mount.IsNotMountPoint(mount.New(""), targetPath)
	if os.IsNotExist(err) {
		notMnt = true
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMnt {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	svr.log.Info("Publishing volume",
		"target", targetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volID,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	var ns corev1.Namespace
	if err := svr.client.Get(ctx, client.ObjectKey{Name: nsName}, &ns); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Failed to query namespace %s: %s", nsName, err.Error()))
	}

	dkName := ns.Labels["oneagent.dynatrace.com/instance"] // TODO(lrgar): replace with constant
	if dkName == "" {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Namespace '%s' doesn't have DynaKube assigned", nsName))
	}

	envID, err := ioutil.ReadFile(filepath.Join(svr.opts.DataDir, fmt.Sprintf("tenant-%s", dkName)))
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("Failed to extract tenant for DynaKube %s: %s", dkName, err.Error()))
	}
	envDir := filepath.Join(svr.opts.DataDir, string(envID))

	for _, dir := range []string{
		filepath.Join(envDir, "log", podUID),
		filepath.Join(envDir, "datastorage", podUID),
	} {
		if err = os.MkdirAll(dir, 0770); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	ver, err := ioutil.ReadFile(filepath.Join(envDir, "version"))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to query agent directory for DynaKube %s: %s", dkName, err.Error()))
	}

	agentDir := filepath.Join(envDir, "bin", string(ver)+"-"+flavor)

	if err := BindMount(
		targetPath,
		Mount{Source: agentDir, Target: targetPath, ReadOnly: true},
		Mount{
			Source: filepath.Join(agentDir, "agent", "conf"),
			Target: filepath.Join(targetPath, "agent", "conf"),
		},
		Mount{
			Source: filepath.Join(envDir, "log", podUID),
			Target: filepath.Join(targetPath, "log"),
		},
		Mount{
			Source: filepath.Join(envDir, "datastorage", podUID),
			Target: filepath.Join(targetPath, "datastorage"),
		},
	); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount OneAgent volume: %s", err.Error()))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (svr *CSIDriverServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// Check arguments
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if err := BindUnmount(
		Mount{Target: filepath.Join(targetPath, "agent", "conf")},
		Mount{Target: filepath.Join(targetPath, "log")},
		Mount{Target: filepath.Join(targetPath, "datastorage")},
		Mount{Target: targetPath},
	); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount volume: %s", err.Error()))
	}

	// Delete the mount point.
	// Does not return error for non-existent path, repeated calls OK for idempotency.
	if err := os.RemoveAll(targetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	svr.log.Info("volume has been unpublished", "targetPath", targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
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

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
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
