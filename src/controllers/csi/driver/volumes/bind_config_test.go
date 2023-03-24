package csivolumes

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/stretchr/testify/assert"
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

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		volumeCfg := &VolumeConfig{
			DynakubeName: testDynakubeName,
		}

		db := metadata.FakeMemoryDB()

		db.InsertDynakube(context.TODO(), metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, "", 0))

		bindCfg, err := NewBindConfig(context.TODO(), db, volumeCfg)

		expected := BindConfig{
			TenantUUID: testTenantUUID,
			Version:    testAgentVersion,
		}
		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, expected, *bindCfg)
	})
}

func TestIsArchiveAvailable(t *testing.T) {
	t.Run(`no version, no digest`, func(t *testing.T) {
		bindCfg := BindConfig{}

		assert.False(t, bindCfg.IsArchiveAvailable())
	})
	t.Run(`version set, no digest`, func(t *testing.T) {
		bindCfg := BindConfig{
			Version: "1.2.3",
		}

		assert.True(t, bindCfg.IsArchiveAvailable())
	})
	t.Run(`no version, digest set`, func(t *testing.T) {
		bindCfg := BindConfig{
			ImageDigest: "sha256:123",
		}

		assert.True(t, bindCfg.IsArchiveAvailable())
	})
}

func TestMetricVersionLabel(t *testing.T) {
	t.Run(`no version, no digest`, func(t *testing.T) {
		bindCfg := BindConfig{}

		assert.Empty(t, bindCfg.MetricVersionLabel())
	})
	t.Run(`version set, no digest`, func(t *testing.T) {
		bindCfg := BindConfig{
			Version: "1.2.3",
		}

		assert.Equal(t, bindCfg.Version, bindCfg.MetricVersionLabel())
	})
	t.Run(`no version, digest set`, func(t *testing.T) {
		bindCfg := BindConfig{
			ImageDigest: "sha256:123",
		}

		assert.Equal(t, bindCfg.ImageDigest, bindCfg.MetricVersionLabel())
	})
}

