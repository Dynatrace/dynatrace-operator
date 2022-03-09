package csivolumes

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

type Publisher interface {
	PublishVolume(ctx context.Context, volumeCfg *VolumeConfig) (*csi.NodePublishVolumeResponse, error)
	UnpublishVolume(ctx context.Context, volumeInfo *VolumeInfo) (*csi.NodeUnpublishVolumeResponse, error)
	CanUnpublishVolume(volumeInfo *VolumeInfo) (bool, error)
}
