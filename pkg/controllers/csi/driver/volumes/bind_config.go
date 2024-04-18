package csivolumes

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BindConfig struct {
	TenantUUID       string
	Version          string
	DynakubeName     string
	MaxMountAttempts int
}

func NewBindConfig(ctx context.Context, access metadata.GormAccess, volumeCfg *VolumeConfig) (*BindConfig, error) {
	tenantConfig, err := access.ReadTenantConfig(metadata.TenantConfig{Name: volumeCfg.DynakubeName})
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("failed to extract tenant for DynaKube %s: %s", volumeCfg.DynakubeName, err.Error()))
	}

	return &BindConfig{
		TenantUUID:       tenantConfig.TenantUUID,
		Version:          tenantConfig.DownloadedCodeModuleVersion,
		DynakubeName:     tenantConfig.Name,
		MaxMountAttempts: int(tenantConfig.MaxFailedMountAttempts),
	}, nil
}

func (cfg BindConfig) IsArchiveAvailable() bool {
	return cfg.Version != ""
}
