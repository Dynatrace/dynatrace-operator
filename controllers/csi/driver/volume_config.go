package csidriver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	podUIDContextKey       = "csi.storage.k8s.io/pod.uid"
	podNamespaceContextKey = "csi.storage.k8s.io/pod.namespace"
)

type volumeConfig struct {
	volumeId   string
	targetPath string
	namespace  string
	podUID     string
}

func parsePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (*volumeConfig, error) {
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

	return &volumeConfig{
		volumeId:   volID,
		targetPath: targetPath,
		namespace:  nsName,
		podUID:     podUID,
	}, nil
}
