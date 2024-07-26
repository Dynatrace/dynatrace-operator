package csivolumes

import (
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PodNameContextKey       = "csi.storage.k8s.io/pod.name"
	NamespaceNameContextKey = "csi.storage.k8s.io/pod.namespace"

	// CSIVolumeAttributeModeField used for identifying the origin of the NodePublishVolume request
	CSIVolumeAttributeModeField         = "mode"
	CSIVolumeAttributeVersionField      = "version"
	CSIVolumeAttributeMountTimeoutField = "timeout"
)

// Represents the basic information about a volume
type VolumeInfo struct {
	VolumeID   string
	TargetPath string
}

// Represents the config needed to mount a volume
type VolumeConfig struct {
	VolumeInfo
	Pod       string
	Namespace string
	Version   string
	Timeout   time.Duration
	Mode      string
}

// Transforms the NodePublishVolumeRequest into a VolumeConfig
func ParseNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (*VolumeConfig, error) {
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

	podName := volCtx[PodNameContextKey]
	if podName == "" {
		return nil, status.Error(codes.InvalidArgument, "No Pod Name included with request")
	}

	namespaceName := volCtx[NamespaceNameContextKey]
	if podName == "" {
		return nil, status.Error(codes.InvalidArgument, "No Namespace Name included with request")
	}

	mode := volCtx[CSIVolumeAttributeModeField]
	if mode == "" {
		return nil, status.Error(codes.InvalidArgument, "No mode attribute included with request")
	}

	version := volCtx[CSIVolumeAttributeVersionField]
	if version == "" {
		return nil, status.Error(codes.InvalidArgument, "No version attribute included with request")
	}

	timeoutStr := volCtx[CSIVolumeAttributeMountTimeoutField]
	if timeoutStr == "" {
		return nil, status.Error(codes.InvalidArgument, "No timeout attribute included with request")
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "timeout attribute does not follow the golang time.Duration format")
	}

	return &VolumeConfig{
		VolumeInfo: VolumeInfo{
			VolumeID:   volID,
			TargetPath: targetPath,
		},
		Pod:       podName,
		Namespace: namespaceName,
		Mode:      mode,
		Version:   version,
		Timeout:   timeout,
	}, nil
}

// Transforms the NodeUnpublishVolumeRequest into a VolumeInfo
func ParseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (*VolumeInfo, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return &VolumeInfo{VolumeID: volumeID, TargetPath: targetPath}, nil
}
