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
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-logr/logr"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	client  client.Client
	log     logr.Logger
	opts    dtcsi.CSIOptions
	fs      afero.Afero
	mounter mount.Interface
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
		<-ctx.Done()
		svr.log.Info("Stopping server")
		server.GracefulStop()
		svr.log.Info("Stopped server")
	}()

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	svr.log.Info("Listening for connections on address", "address", listener.Addr())

	_ = server.Serve(listener)

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

	if isMounted, err := isMounted(mount.New(""), volumeCfg.targetPath); err != nil {
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
		//TODO _ = svr.umountOneAgent("???")
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount oneagent volume: %s", err))
	}

	if err := svr.makePodToVersionReference(bindCfg.podToVersionReferenceDir, volumeCfg.podUID, volumeCfg.volumeId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to create pod to version reference: %s", err))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

//TODO test
func (svr *CSIDriverServer) makePodToVersionReference(podToVersionReferenceDir string, podUID string, volumeID string) error {
	podToVersionReference := filepath.Join(podToVersionReferenceDir, podUID)
	if err := svr.fs.WriteFile(podToVersionReference, nil, 0770); err != nil {
		return err
	}

	if err := svr.fs.Fs.(afero.Linker).SymlinkIfPossible(podToVersionReference, filepath.Join(svr.opts.RootDir, dtcsi.GarbageCollectionPath, volumeID)); err != nil {
		return err
	}

	return nil
}

func (svr *CSIDriverServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// Check arguments
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if err := svr.umountOneAgent(targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount oneagent volume: %s", err.Error()))
	}

	volumeToPodReference := filepath.Join(svr.opts.RootDir, dtcsi.GarbageCollectionPath, volumeID)

	podToVersionReference, err := svr.fs.Fs.(afero.LinkReader).ReadlinkIfPossible(volumeToPodReference)
	if err != nil && !os.IsNotExist(err) {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to read volume to pod SymLink for garbage collector - error: %s", err))
	} else if !os.IsNotExist(err) {
		if err = svr.fs.Remove(podToVersionReference); err != nil && !os.IsNotExist(err) {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to remove pod to version reference file for garbage collector - error: %s", err))
		}
		if err = svr.fs.Remove(volumeToPodReference); err != nil && !os.IsNotExist(err) {
			return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to remove volume to pod SymLink for garbage collector - error: %s", err))
		}
	}

	// Delete the mount point.
	// Does not return error for non-existent path, repeated calls OK for idempotency.
	if err := svr.fs.RemoveAll(targetPath); err != nil {
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

func (svr *CSIDriverServer) mountOneAgent(bindCfg *bindConfig, volumeCfg *volumeConfig) error {
	//TODO const-ify run/mapped/var/work strings
	//TODO probably give all directories proper names and create methods for them
	mappedDir := filepath.Join(bindCfg.envDir, "run", volumeCfg.podUID, "mapped")
	_ = svr.fs.MkdirAll(mappedDir, os.ModePerm)

	upperDir := filepath.Join(bindCfg.envDir, "run", volumeCfg.podUID, "var")
	_ = svr.fs.MkdirAll(upperDir, os.ModePerm)

	workDir := filepath.Join(bindCfg.envDir, "run", volumeCfg.podUID, "work")
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
		return err
	}

	return nil
}

func (svr *CSIDriverServer) umountOneAgent(targetPath string) error {
	if err := svr.mounter.Unmount(targetPath); err != nil {
		return err
	}

	//TODO svr.mounter.Unmount(mappedDir)

	return nil
}
