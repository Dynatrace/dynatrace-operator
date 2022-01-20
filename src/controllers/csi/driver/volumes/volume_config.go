package csivolumes

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PodNameContextKey      = "csi.storage.k8s.io/pod.name"
	PodNamespaceContextKey = "csi.storage.k8s.io/pod.namespace"
)

type VolumeInfo struct {
	VolumeId   string
	TargetPath string
}

type VolumeConfig struct {
	VolumeInfo
	Namespace string
	PodName   string
}

func ParsePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (*VolumeConfig, error) {
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

	nsName := volCtx[PodNamespaceContextKey]
	if nsName == "" {
		return nil, status.Error(codes.InvalidArgument, "No namespace included with request")
	}

	podName := volCtx[PodNameContextKey]
	if podName == "" {
		return nil, status.Error(codes.InvalidArgument, "No Pod Name included with request")
	}

	return &VolumeConfig{
		VolumeInfo: VolumeInfo{
			VolumeId:   volID,
			TargetPath: targetPath,
		},
		Namespace: nsName,
		PodName:   podName,
	}, nil
}

func ParseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (*VolumeInfo, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return &VolumeInfo{volumeID, targetPath}, nil
}
