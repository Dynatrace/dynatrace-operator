package csiprovisioner

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testTenantUUID          = "zib123"
	testProcessModuleConfig = dtclient.ProcessModuleConfig{
		Revision: 3,
		Properties: []dtclient.ProcessModuleProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test3",
			},
		},
	}
	testProcessModuleConfigCache = processModuleConfigCache{
		ProcessModuleConfig: &dtclient.ProcessModuleConfig{
			Revision: 1,
			Properties: []dtclient.ProcessModuleProperty{
				{
					Section: "test",
					Key:     "test",
					Value:   "test1",
				},
			},
		},
		Hash: "asd",
	}
)

func prepTestFsCache(fs afero.Fs, content []byte) {
	path := metadata.PathResolver{}
	cache, _ := fs.OpenFile(path.AgentRuxitProcResponseCache(testTenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	cache.Write(content)
}

func isCacheExisting(fs afero.Fs) error {
	path := metadata.PathResolver{}
	_, err := fs.Open(path.AgentRuxitProcResponseCache(testTenantUUID))
	return err
}

func TestGetProcessModuleConfig(t *testing.T) {
	var emptyResponse *dtclient.ProcessModuleConfig
	t.Run(`no cache + no revision (dry run)`, func(t *testing.T) {
		var defaultHash string
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := provisioner.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfig, *response)
		assert.Equal(t, defaultHash, storedHash)
	})
	t.Run(`cache + latest revision (cached run)`, func(t *testing.T) {
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
		memFs := afero.NewMemMapFs()
		content, _ := json.Marshal(testProcessModuleConfigCache)
		prepTestFsCache(memFs, content)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", testProcessModuleConfigCache.Revision).
			Return(&testProcessModuleConfig, nil)
		provisioner := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := provisioner.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfig, *response)
		assert.Equal(t, testProcessModuleConfigCache.Hash, storedHash)
	})
}

func TestReadProcessModuleConfigCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	content, _ := json.Marshal(testProcessModuleConfigCache)
	prepTestFsCache(memFs, content)
	provisioner := &OneAgentProvisioner{
		fs: memFs,
	}

	cache, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testProcessModuleConfigCache, *cache)
}

func TestReadInvalidProcessModuleConfigCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	content := []byte("this is invalid json")
	prepTestFsCache(memFs, content)
	provisioner := &OneAgentProvisioner{
		fs: memFs,
	}

	_, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
	assert.Error(t, isCacheExisting(memFs))
	assert.Error(t, err)
}

func TestWriteProcessModuleConfigCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	provisioner := &OneAgentProvisioner{
		fs: memFs,
	}

	err := provisioner.writeProcessModuleConfigCache(testTenantUUID, &testProcessModuleConfigCache)

	require.Nil(t, err)
	cache, err := provisioner.readProcessModuleConfigCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testProcessModuleConfigCache, *cache)
	assert.NoError(t, isCacheExisting(memFs))
}
