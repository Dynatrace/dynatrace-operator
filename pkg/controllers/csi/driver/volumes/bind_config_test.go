package csivolumes

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDynakubeName = "a-dynakube"
	testTenantUUID   = "a-tenant-uuid"
	testAgentVersion = "1.2-3"
)

func TestNewBindConfig(t *testing.T) {
	t.Run(`no dynakube in storage`, func(t *testing.T) {
		volumeCfg := &VolumeConfig{
			DynakubeName: testDynakubeName,
		}

		bindCfg, err := NewBindConfig(context.TODO(), metadata.FakeMemoryDB(), volumeCfg)

		require.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		volumeCfg := &VolumeConfig{
			DynakubeName: testDynakubeName,
		}

		db := metadata.FakeMemoryDB()

		tenantConfig := metadata.TenantConfig{
			Name:                        testDynakubeName,
			TenantUUID:                  testTenantUUID,
			DownloadedCodeModuleVersion: testAgentVersion,
			MaxFailedMountAttempts:      1,
		}
		db.CreateTenantConfig(&tenantConfig)

		bindCfg, err := NewBindConfig(context.Background(), db, volumeCfg)

		expected := BindConfig{
			TenantUUID:       testTenantUUID,
			Version:          testAgentVersion,
			DynakubeName:     testDynakubeName,
			MaxMountAttempts: 1,
		}

		require.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, expected, *bindCfg)
	})
}

func TestIsArchiveAvailable(t *testing.T) {
	t.Run(`no version`, func(t *testing.T) {
		bindCfg := BindConfig{}

		assert.False(t, bindCfg.IsArchiveAvailable())
	})
	t.Run(`version set`, func(t *testing.T) {
		bindCfg := BindConfig{
			Version: "1.2.3",
		}

		assert.True(t, bindCfg.IsArchiveAvailable())
	})
}
