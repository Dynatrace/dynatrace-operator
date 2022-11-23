package csivolumes

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BindConfig struct {
	TenantUUID       string
	Version          string
	ImageDigest      string
	MaxMountAttempts int
}

func NewBindConfig(ctx context.Context, access metadata.Access, volumeCfg *VolumeConfig) (*BindConfig, error) {
	dynakube, err := access.GetDynakube(ctx, volumeCfg.DynakubeName)
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("failed to extract tenant for DynaKube %s: %s", volumeCfg.DynakubeName, err.Error()))
	}
	if dynakube == nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("dynakube (%s) is missing from metadata database", volumeCfg.DynakubeName))
	}
	return &BindConfig{
		TenantUUID:       dynakube.TenantUUID,
		Version:          dynakube.LatestVersion,
		ImageDigest:      dynakube.ImageDigest,
		MaxMountAttempts: dynakube.MaxFailedMountAttempts,
	}, nil
}
