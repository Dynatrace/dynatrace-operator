package csiprovisioner

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantUUID = "zib123"
)

func createTestProcessModuleConfig(revision uint) *dtclient.ProcessModuleConfig {
	return &dtclient.ProcessModuleConfig{
		Revision: revision,
		Properties: []dtclient.ProcessModuleProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test3",
			},
		},
	}
}

func createTestProcessModuleConfigCache(revision uint) processModuleConfigCache {
	return processModuleConfigCache{
		ProcessModuleConfig: createTestProcessModuleConfig(revision),
		Hash:                strconv.FormatUint(uint64(revision), 10),
	}
}

func prepTestFsCache(fs afero.Fs, content []byte) {
	path := metadata.PathResolver{}
	cache, _ := fs.OpenFile(path.AgentRuxitProcResponseCache(testTenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	cache.Write(content)
}

func isCacheExisting(fs afero.Fs) bool {
	path := metadata.PathResolver{}
	_, err := fs.Open(path.AgentRuxitProcResponseCache(testTenantUUID))
	return err == nil
}

func TestGetProcessModuleConfig(t *testing.T) {
	var emptyResponse *dtclient.ProcessModuleConfig
	t.Run(`no cache + no revision (dry run)`, func(t *testing.T) {
		var defaultHash string
		testProcessModuleConfig := createTestProcessModuleConfig(3)
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", uint(0)).
			Return(testProcessModuleConfig, nil)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := provisioner.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, *testProcessModuleConfig, *response)
		assert.Equal(t, defaultHash, storedHash)
	})
	t.Run(`cache + latest revision (cached run)`, func(t *testing.T) {
		var revision uint = 3
		testProcessModuleConfigCache := createTestProcessModuleConfigCache(revision)
		memFs := afero.NewMemMapFs()
		content, _ := json.Marshal(testProcessModuleConfigCache)
		prepTestFsCache(memFs, content)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", testProcessModuleConfigCache.Revision).
			Return(emptyResponse, nil)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := provisioner.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfigCache.ProcessModuleConfig, response)
		assert.Equal(t, testProcessModuleConfigCache.Hash, storedHash)
	})
	t.Run(`cache + old revision (outdated cache should be ignored)`, func(t *testing.T) {
		var revision uint = 3
		testProcessModuleConfig := createTestProcessModuleConfig(revision)
		testProcessModuleConfigCache := createTestProcessModuleConfigCache(revision)

		memFs := afero.NewMemMapFs()
		content, _ := json.Marshal(testProcessModuleConfigCache)
		prepTestFsCache(memFs, content)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", testProcessModuleConfigCache.Revision).
			Return(testProcessModuleConfig, nil)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := provisioner.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, *testProcessModuleConfig, *response)
		assert.Equal(t, testProcessModuleConfigCache.Hash, storedHash)
	})
}

func TestReadProcessModuleConfigCache(t *testing.T) {
	t.Run(`read cache successful`, func(t *testing.T) {
		var revision uint = 3
		testProcessModuleConfigCache := createTestProcessModuleConfigCache(revision)
		memFs := afero.NewMemMapFs()
		content, _ := json.Marshal(testProcessModuleConfigCache)
		prepTestFsCache(memFs, content)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		cache, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfigCache, *cache)
	})
	t.Run(`read invalid json from cache`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		content := []byte("this is invalid json")
		prepTestFsCache(memFs, content)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		_, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
		assert.False(t, isCacheExisting(memFs))
		assert.Error(t, err)
	})
	t.Run(`write cache successful`, func(t *testing.T) {
		var revision uint = 3
		testProcessModuleConfigCache := createTestProcessModuleConfigCache(revision)
		memFs := afero.NewMemMapFs()
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		err := provisioner.writeProcessModuleConfigCache(testTenantUUID, &testProcessModuleConfigCache)

		require.Nil(t, err)
		cache, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfigCache, *cache)
		assert.True(t, isCacheExisting(memFs))
	})
}
