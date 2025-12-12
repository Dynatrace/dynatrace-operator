package csivolumes

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
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
func ParseNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (VolumeConfig, error) {
	volumeConfig := VolumeConfig{}

	volumeInfo, err := newVolumeInfo(req)
	volumeConfig.VolumeInfo = volumeInfo

	if err != nil {
		return volumeConfig, err
	}

	if req.GetVolumeCapability() == nil {
		return volumeConfig, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}

	if req.GetVolumeCapability().GetBlock() != nil {
		return volumeConfig, status.Error(codes.InvalidArgument, "cannot have block access type")
	}

	if req.GetVolumeCapability().GetMount() == nil {
		return volumeConfig, status.Error(codes.InvalidArgument, "expecting to have mount access type")
	}

	volCtx := req.GetVolumeContext()
	if volCtx == nil {
		return volumeConfig, status.Error(codes.InvalidArgument, "Publish context missing in request")
	}

	podName := volCtx[PodNameContextKey]
	if podName == "" {
		return volumeConfig, status.Error(codes.InvalidArgument, "No Pod Name included in request")
	}

	volumeConfig.PodName = podName

	podNamespace := volCtx[PodNamespaceContextKey]
	if podNamespace == "" {
		return volumeConfig, status.Error(codes.InvalidArgument, "No Pod Namespace included in request")
	}

	volumeConfig.PodNamespace = podNamespace

	mode := volCtx[CSIVolumeAttributeModeField]
	if mode == "" {
		return volumeConfig, status.Error(codes.InvalidArgument, "No mode attribute included in request")
	}

	volumeConfig.Mode = mode

	dynakubeName := volCtx[CSIVolumeAttributeDynakubeField]
	if dynakubeName == "" {
		return volumeConfig, status.Error(codes.InvalidArgument, "No dynakube attribute included in request")
	}

	volumeConfig.DynakubeName = dynakubeName

	retryTimeoutValue := volCtx[CSIVolumeAttributeRetryTimeout]
	if retryTimeoutValue == "" {
		retryTimeoutValue = exp.DefaultCSIMaxMountTimeout
	}

	retryTimeout, err := time.ParseDuration(retryTimeoutValue)
	if err != nil {
		return volumeConfig, status.Error(codes.InvalidArgument, "The retryTimeout attribute has incorrect format")
	}

	volumeConfig.RetryTimeout = retryTimeout

	return volumeConfig, nil
}

// Transforms the NodeUnpublishVolumeRequest into a VolumeInfo
func ParseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (VolumeInfo, error) {
	return newVolumeInfo(req)
}

type baseRequest interface {
	GetVolumeId() string
	GetTargetPath() string
}

func newVolumeInfo(req baseRequest) (VolumeInfo, error) {
	info := VolumeInfo{}

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return info, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	info.VolumeID = volumeID

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return info, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	info.TargetPath = targetPath

	return info, nil
}
