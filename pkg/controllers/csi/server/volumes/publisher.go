package csivolumes

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

type Publisher interface {
	PublishVolume(ctx context.Context, volumeCfg *VolumeConfig) (*csi.NodePublishVolumeResponse, error)
}
