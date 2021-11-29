package csiprovisioner

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testTenantUUID          = "zib123"
	testVersion             = "v123"
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

	testRuxitConf = `
[general]
key value
`
)

func prepTestFsCache(fs afero.Fs) {
	testCache, _ := json.Marshal(testProcessModuleConfigCache)
	path := metadata.PathResolver{}
	cache, _ := fs.OpenFile(path.AgentRuxitProcResponseCache(testTenantUUID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	cache.Write(testCache)
}

func TestGetProcessModuleConfig(t *testing.T) {
	var emptyResponse *dtclient.ProcessModuleConfig
	t.Run(`no cache + no revision (dry run)`, func(t *testing.T) {
		var defaultHash string
		memFs := afero.NewMemMapFs()
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := r.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfig, *response)
		assert.Equal(t, defaultHash, storedHash)
	})
	t.Run(`cache + latest revision (cached run)`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		prepTestFsCache(memFs)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", testProcessModuleConfigCache.Revision).
			Return(emptyResponse, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := r.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfigCache.ProcessModuleConfig, response)
		assert.Equal(t, testProcessModuleConfigCache.Hash, storedHash)
	})
	t.Run(`cache + old revision (outdated cache should be ignored)`, func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		prepTestFsCache(memFs)
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetProcessModuleConfig", testProcessModuleConfigCache.Revision).
			Return(&testProcessModuleConfig, nil)
		r := &OneAgentProvisioner{
			fs: memFs,
		}

		response, storedHash, err := r.getProcessModuleConfig(mockClient, testTenantUUID)

		require.Nil(t, err)
		assert.Equal(t, testProcessModuleConfig, *response)
		assert.Equal(t, testProcessModuleConfigCache.Hash, storedHash)
	})
}

func TestReadProcessModuleConfigCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	prepTestFsCache(memFs)
	r := &OneAgentProvisioner{
		fs: memFs,
	}

	cache, err := r.readProcessModuleConfigCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testProcessModuleConfigCache, *cache)
}

func TestWriteProcessModuleConfigCache(t *testing.T) {
	memFs := afero.NewMemMapFs()
	r := &OneAgentProvisioner{
		fs: memFs,
	}

	err := r.writeProcessModuleConfigCache(testTenantUUID, &testProcessModuleConfigCache)

	require.Nil(t, err)
	cache, err := r.readProcessModuleConfigCache(testTenantUUID)
	require.Nil(t, err)
	assert.Equal(t, testProcessModuleConfigCache, *cache)
}

func prepTestConfFs(fs afero.Fs) {
	path := metadata.PathResolver{}
	fs.MkdirAll(filepath.Base(path.AgentProcessModuleConfigForVersion(testTenantUUID, testVersion)), 0755)
	usedConf, _ := fs.OpenFile(path.AgentProcessModuleConfigForVersion(testTenantUUID, testVersion), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	usedConf.WriteString(testRuxitConf)
}

func assertTestConf(t *testing.T, fs afero.Fs, path, expected string) {
	file, err := fs.Open(path)
	require.Nil(t, err)
	content, err := ioutil.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, expected, string(content))
}

func TestUpdateProcessModuleConfig(t *testing.T) {
	path := metadata.PathResolver{}
	memFs := afero.NewMemMapFs()
	prepTestConfFs(memFs)
	agentConfig := &installAgentConfig{
		fs:     memFs,
		dk:     &dynatracev1beta1.DynaKube{},
		logger: logger.NewDTLogger(),
	}
	expectedUsed := `
[general]
key value

[test]
test test3
`

	agentConfig.updateProcessModuleConfig(testVersion, testTenantUUID, &testProcessModuleConfig)

	assertTestConf(t, memFs, path.AgentProcessModuleConfigForVersion(testTenantUUID, testVersion), expectedUsed)
	assertTestConf(t, memFs, path.SourceAgentProcessModuleConfigForVersion(testTenantUUID, testVersion), testRuxitConf)
}

func TestCheckProcessModuleConfigCopy(t *testing.T) {
	memFs := afero.NewMemMapFs()
	path := metadata.PathResolver{}
	prepTestConfFs(memFs)
	agentConfig := &installAgentConfig{
		fs: memFs,
	}
	sourcePath := path.SourceAgentProcessModuleConfigForVersion(testTenantUUID, testVersion)
	destPath := path.AgentProcessModuleConfigForVersion(testTenantUUID, testVersion)

	agentConfig.checkProcessModuleConfigCopy(sourcePath, destPath)

	assertTestConf(t, memFs, sourcePath, testRuxitConf)
}

func TestAddHostGroup(t *testing.T) {
	t.Run(`dk with hostGroup, no api`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Args: []string{
								"--set-host-group=test",
							},
						},
					},
				},
			},
		}
		emptyResponse := dtclient.ProcessModuleConfig{}
		result := addHostGroup(dk, &emptyResponse)
		assert.NotNil(t, result)
		assert.Equal(t, "test", result.ToMap()["general"]["hostGroup"])
	})
	t.Run(`dk with hostGroup, api present`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							Args: []string{
								"--set-host-group=test",
							},
						},
					},
				},
			},
		}
		pmc := dtclient.ProcessModuleConfig{
			Properties: []dtclient.ProcessModuleProperty{
				{
					Section: "general",
					Key:     "other",
					Value:   "other",
				},
			},
		}
		result := addHostGroup(dk, &pmc)
		assert.NotNil(t, result)
		assert.Len(t, result.ToMap()["general"], 2)
		assert.Equal(t, "test", result.ToMap()["general"]["hostGroup"])
	})
	t.Run(`dk without hostGroup`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
					},
				},
			},
		}
		pmc := dtclient.ProcessModuleConfig{
			Properties: []dtclient.ProcessModuleProperty{
				{
					Section: "general",
					Key:     "other",
					Value:   "other",
				},
			},
		}
		result := addHostGroup(dk, &pmc)
		assert.NotNil(t, result)
		assert.Equal(t, *result, pmc)
	})
	t.Run(`dk without hostGroup, remove previous hostgroup`, func(t *testing.T) {
		dk := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
					},
				},
			},
		}
		pmc := dtclient.ProcessModuleConfig{
			Properties: []dtclient.ProcessModuleProperty{
				{
					Section: "general",
					Key:     "hostGroup",
					Value:   "other",
				},
			},
		}
		result := addHostGroup(dk, &pmc)
		assert.NotNil(t, result)
		assert.Len(t, pmc.Properties, 0)
	})
}
