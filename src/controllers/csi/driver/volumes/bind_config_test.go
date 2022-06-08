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

		db.InsertDynakube(metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion, false))

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
