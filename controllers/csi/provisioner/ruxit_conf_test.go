package csiprovisioner

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testTenantUUID        = "zib123"
	testVersion           = "v123"
	testRuxitProcResponse = dtclient.RuxitProcResponse{
		Revision: 3,
		Properties: []dtclient.RuxitProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test3",
			},
		},
	}
	testRuxitProcResponseCache = dtclient.RuxitProcResponse{
		Revision: 1,
		Properties: []dtclient.RuxitProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test1",
			},
		},
	}

	testRuxitConf = `
[general]
key value
`
)

func prepTestFsCache(fs afero.Fs) {
	testCache, _ := json.Marshal(testRuxitProcResponseCache)
	path := metadata.PathResolver{}
	cache, _ := fs.OpenFile(path.AgentRuxitProcResponseCache(testTenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	cache.Write(testCache)
}

func TestGetRuxitProcResponse(t *testing.T) {
	var emptyResponse *dtclient.RuxitProcResponse
	t.Run(`no cache + no revision (dry run)`, func(t *testing.T) {
		var defaultRevision uint
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetRuxitProcConf", defaultRevision).
			Return(&testRuxitProcResponse, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, lastRevision, err := r.getRuxitProcResponse(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testRuxitProcResponse, *response)
		assert.Equal(t, defaultRevision, lastRevision)
	})
	t.Run(`cache + latest revision (cached run)`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		prepTestFsCache(memFs)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetRuxitProcConf", testRuxitProcResponseCache.Revision).
			Return(emptyResponse, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, lastRevision, err := r.getRuxitProcResponse(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testRuxitProcResponseCache, *response)
		assert.Equal(t, testRuxitProcResponseCache.Revision, lastRevision)
	})
	t.Run(`cache + old revision (outdated cache should be ignored)`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		prepTestFsCache(memFs)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetRuxitProcConf", testRuxitProcResponseCache.Revision).
			Return(&testRuxitProcResponse, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, lastRevision, err := r.getRuxitProcResponse(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testRuxitProcResponse, *response)
		assert.Equal(t, testRuxitProcResponseCache.Revision, lastRevision)
	})
}

func TestReadRuxitCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	prepTestFsCache(memFs)
	r := &OneAgentProvisioner{
		fs: memFs,
	}

	cache, err := r.readRuxitCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testRuxitProcResponseCache, *cache)
}

func TestWriteRuxitCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	r := &OneAgentProvisioner{
		fs: memFs,
	}

	err := r.writeRuxitCache(testTenantUUID, &testRuxitProcResponseCache)

	require.Nil(t, err)
	cache, err := r.readRuxitCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testRuxitProcResponseCache, *cache)
}

func prepTestConfFs(fs afero.Fs) {
	path := metadata.PathResolver{}
	fs.MkdirAll(filepath.Base(path.AgentRuxitConfForVersion(testTenantUUID, testVersion)), 0755)
	usedConf, _ := fs.OpenFile(path.AgentRuxitConfForVersion(testTenantUUID, testVersion), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	usedConf.WriteString(testRuxitConf)
}

func assertTestConf(t *testing.T, fs afero.Fs, path, expected string) {
	file, err := fs.Open(path)
	require.Nil(t, err)
	content, err := ioutil.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, expected, string(content))
}

func TestUpdateRuxitConf(t *testing.T) {
	path := metadata.PathResolver{}
	memFs := afero.NewMemMapFs()
	prepTestConfFs(memFs)
	agentConfig := &installAgentConfig{
		fs:     memFs,
		logger: logger.NewDTLogger(),
	}
	expectedUsed := `
[general]
key value

[test]
test test3
`

	agentConfig.updateRuxitConf(testVersion, testTenantUUID, &testRuxitProcResponse)

	assertTestConf(t, memFs, path.AgentRuxitConfForVersion(testTenantUUID, testVersion), expectedUsed)
	assertTestConf(t, memFs, path.SourceAgentRuxitConfForVersion(testTenantUUID, testVersion), testRuxitConf)
}

func TestCheckRuxitConfCopy(t *testing.T) {
	memFs := afero.NewMemMapFs()
	path := metadata.PathResolver{}
	prepTestConfFs(memFs)
	agentConfig := &installAgentConfig{
		fs: memFs,
	}
	sourcePath := path.SourceAgentRuxitConfForVersion(testTenantUUID, testVersion)
	destPath := path.AgentRuxitConfForVersion(testTenantUUID, testVersion)

	agentConfig.checkRuxitConfCopy(sourcePath, destPath)

	assertTestConf(t, memFs, sourcePath, testRuxitConf)
}
