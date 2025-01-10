package csivolumes

import (
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PodNameContextKey      = "csi.storage.k8s.io/pod.name"
	PodNamespaceContextKey = "csi.storage.k8s.io/pod.namespace"

	// CSIVolumeAttributeModeField used for identifying the origin of the NodePublishVolume request
	CSIVolumeAttributeModeField     = "mode"
	CSIVolumeAttributeDynakubeField = "dynakube"
	CSIVolumeAttributeRetryTimeout  = "retryTimeout"
)

// Represents the basic information about a volume
type VolumeInfo struct {
	VolumeID   string
	TargetPath string
}

// Represents the config needed to mount a volume
type VolumeConfig struct {
	VolumeInfo
	PodName      string
	PodNamespace string
	Mode         string
	DynakubeName string
	RetryTimeout time.Duration
}

// Transforms the NodePublishVolumeRequest into a VolumeConfig
func ParseNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (*VolumeConfig, error) {
	volumeInfo, err := newVolumeInfo(req)
	if err != nil {
		return nil, err
	}

	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
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
		return nil, status.Error(codes.InvalidArgument, "No Pod Name included in request")
	}

	podNamespace := volCtx[PodNamespaceContextKey]
	if podNamespace == "" {
		return nil, status.Error(codes.InvalidArgument, "No Pod Namespace included in request")
	}

	mode := volCtx[CSIVolumeAttributeModeField]
	if mode == "" {
		return nil, status.Error(codes.InvalidArgument, "No mode attribute included in request")
	}

	dynakubeName := volCtx[CSIVolumeAttributeDynakubeField]
	if dynakubeName == "" {
		return nil, status.Error(codes.InvalidArgument, "No dynakube attribute included in request")
	}

	retryTimeoutValue := volCtx[CSIVolumeAttributeRetryTimeout]
	if retryTimeoutValue == "" {
		return nil, status.Error(codes.InvalidArgument, "No retryTimeout attribute included in request")
	}

	retryTimeout, err := time.ParseDuration(retryTimeoutValue)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The retryTimeout attribute has incorrect format")
	}

	return &VolumeConfig{
		VolumeInfo:   *volumeInfo,
		PodName:      podName,
		PodNamespace: podNamespace,
		Mode:         mode,
		DynakubeName: dynakubeName,
		RetryTimeout: retryTimeout,
	}, nil
}

// Transforms the NodeUnpublishVolumeRequest into a VolumeInfo
func ParseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (*VolumeInfo, error) {
	return newVolumeInfo(req)
}

type baseRequest interface {
	GetVolumeId() string
	GetTargetPath() string
}

func newVolumeInfo(req baseRequest) (*VolumeInfo, error) {
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
