package standalone

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/installer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testProcessModuleConfig = dtclient.ProcessModuleConfig{
	Revision: 0,
	Properties: []dtclient.ProcessModuleProperty{
		{
			Section: "test",
			Key:     "test",
			Value:   "test",
		},
	},
}

func TestNewRunner(t *testing.T) {
	fs := prepTestFs(t)
	t.Run(`create runner with oneagent and data-ingest injection`, func(t *testing.T) {
		resetEnv := prepCombinedTestEnv(t)
		runner, err := NewRunner(fs)
		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, runner)
		assert.NotNil(t, runner.fs)
		assert.NotNil(t, runner.env)
		assert.NotNil(t, runner.dtclient)
		assert.NotNil(t, runner.config)
		assert.NotNil(t, runner.installer)
		assert.Empty(t, runner.hostTenant)
	})
	t.Run(`create runner with only oneagent`, func(t *testing.T) {
		resetEnv := prepOneAgentTestEnv(t)
		runner, err := NewRunner(fs)
		resetEnv()

		require.NoError(t, err)
		assert.NotNil(t, runner.fs)
		assert.NotNil(t, runner.env)
		assert.NotNil(t, runner.dtclient)
		assert.NotNil(t, runner.config)
		assert.NotNil(t, runner.installer)
		assert.Empty(t, runner.hostTenant)
	})
	t.Run(`create runner with only data-ingest injection`, func(t *testing.T) {
		resetEnv := prepDataIngestTestEnv(t)
		runner, err := NewRunner(fs)
		resetEnv()

		require.NoError(t, err)
		assert.NotNil(t, runner.fs)
		assert.NotNil(t, runner.env)
		assert.Nil(t, runner.dtclient)
		assert.Nil(t, runner.config)
		assert.Nil(t, runner.installer)
		assert.Empty(t, runner.hostTenant)
	})
}

func TestConsumeErrorIfNecessary(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run(`no error thrown`, func(t *testing.T) {
		runner.env.CanFail = false
		err := runner.Run()
		assert.Nil(t, err)
	})
	t.Run(`error thrown, but consume error`, func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.CanFail = false
		err := runner.Run()
		assert.Nil(t, err)
	})
	t.Run(`error thrown, but don't consume error`, func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.CanFail = true
		err := runner.Run()
		assert.NotNil(t, err)
	})
}

func TestSetHostTenant(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run(`fail due to missing node`, func(t *testing.T) {
		runner.config.HasHost = true

		err := runner.setHostTenant()

		require.Error(t, err)
	})
	t.Run(`set hostTenant to node`, func(t *testing.T) {
		runner.config.HasHost = true
		runner.env.K8NodeName = testNodeName

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, runner.hostTenant)
	})
	t.Run(`set hostTenant to empty`, func(t *testing.T) {
		runner.config.HasHost = false

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, config.AgentNoHostTenant, runner.hostTenant)
	})
}

func TestInstallOneAgent(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run(`happy install`, func(t *testing.T) {
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", config.AgentBinDirMount, &testProcessModuleConfig).
			Return(nil)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", config.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.NoError(t, err)
	})
	t.Run(`sad install -> install fail`, func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", config.AgentBinDirMount).
			Return(false, fmt.Errorf("BOOM"))

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run(`sad install -> ruxitagent update fail`, func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", config.AgentBinDirMount, &testProcessModuleConfig).
			Return(fmt.Errorf("BOOM"))
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", config.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run(`sad install -> ruxitagent endpoint fail`, func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", config.AgentBinDirMount, &testProcessModuleConfig).
			Return(nil)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", config.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.Error(t, err)
	})
}
func TestRun(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false
	runner.env.OneAgentInjected = true
	runner.env.DataIngestInjected = true
	runner.dtclient.(*dtclient.MockDynatraceClient).
		On("GetProcessModuleConfig", uint(0)).
		Return(&testProcessModuleConfig, nil)
	runner.installer.(*installer.InstallerMock).
		On("UpdateProcessModuleConfig", config.AgentBinDirMount, &testProcessModuleConfig).
		Return(nil)

	t.Run(`no install, just config generation`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.Mode = config.AgentCsiMode

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)

	})
	t.Run(`install + config generation`, func(t *testing.T) {
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", config.AgentBinDirMount).
			Return(true, nil)
		runner.fs = afero.NewMemMapFs()
		runner.env.Mode = config.AgentInstallerMode

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)

	})
}

func TestConfigureInstallation(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run(`create all config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = true
		runner.env.DataIngestInjected = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)

	})
	t.Run(`create only container confs`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = true
		runner.env.DataIngestInjected = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesNotExists(t, *runner)

	})
	t.Run(`create only enrichment file`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = false
		runner.env.DataIngestInjected = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesNotExists(t, *runner)
		// enrichemt
		assertIfEnrichmentFilesExists(t, *runner)

	})
}

func TestCreateContainerConfigurationFiles(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run(`create config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.createContainerConfigurationFiles()

		require.NoError(t, err)
		for _, container := range runner.env.Containers {
			assertIfFileExists(t,
				runner.fs,
				filepath.Join(
					config.AgentShareDirMount,
					fmt.Sprintf(config.AgentContainerConfFilenameTemplate, container.Name)))
		}
		// TODO: Check content ?
	})
}

func TestSetLDPreload(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run(`create ld preload file`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.setLDPreload()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				config.AgentShareDirMount,
				config.LdPreloadFilename))
		// TODO: Check content ?
	})
}

func TestEnrichMetadata(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run(`create enrichment files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.enrichMetadata()

		require.NoError(t, err)
		assertIfEnrichmentFilesExists(t, *runner)
		// TODO: Check content ?
	})
}

func TestPropagateTLSCert(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run(`create tls custom.pem`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.propagateTLSCert()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(config.AgentShareDirMount, "custom.pem"))
	})
}

func TestWriteCurlOptions(t *testing.T) {
	filesystem := afero.NewMemMapFs()
	runner := Runner{
		config: &SecretConfig{InitialConnectRetry: 30},
		env:    &environment{OneAgentInjected: true},
		fs:     filesystem,
	}

	err := runner.configureInstallation()

	assert.NoError(t, err)

	exists, err := afero.Exists(filesystem, "/mnt/share/curl_options.conf")

	assert.NoError(t, err)
	assert.True(t, exists)
}

func creatTestRunner(t *testing.T) *Runner {
	fs := prepTestFs(t)
	resetEnv := prepCombinedTestEnv(t)

	runner, err := NewRunner(fs)
	resetEnv()
	require.NoError(t, err)
	require.NotNil(t, runner)
	return runner
}

func createMockedRunner(t *testing.T) *Runner {
	runner := creatTestRunner(t)
	runner.installer = &installer.InstallerMock{}
	runner.dtclient = &dtclient.MockDynatraceClient{}
	return runner
}

func assertIfAgentFilesExists(t *testing.T, runner Runner) {
	// container confs
	for _, container := range runner.env.Containers {
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				config.AgentShareDirMount,
				fmt.Sprintf(config.AgentContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(config.AgentShareDirMount, config.LdPreloadFilename))
	// tls cert
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(config.AgentShareDirMount, "custom.pem"))

}

func assertIfEnrichmentFilesExists(t *testing.T, runner Runner) {
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(
			config.EnrichmentMountPath,
			fmt.Sprintf(config.EnrichmentFilenameTemplate, "json")))
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(
			config.EnrichmentMountPath,
			fmt.Sprintf(config.EnrichmentFilenameTemplate, "properties")))

}

func assertIfAgentFilesNotExists(t *testing.T, runner Runner) {
	// container confs
	for _, container := range runner.env.Containers {
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				config.AgentShareDirMount,
				fmt.Sprintf(config.AgentContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(config.AgentShareDirMount, config.LdPreloadFilename))
	// tls cert
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(config.AgentShareDirMount, "custom.pem"))

}

func assertIfEnrichmentFilesNotExists(t *testing.T, runner Runner) {
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(
			config.EnrichmentMountPath,
			fmt.Sprintf(config.EnrichmentFilenameTemplate, "json")))
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(
			config.EnrichmentMountPath,
			fmt.Sprintf(config.EnrichmentFilenameTemplate, "properties")))

}

func assertIfFileExists(t *testing.T, fs afero.Fs, path string) {
	fileInfo, err := fs.Stat(path)
	assert.NoError(t, err)
	assert.NotNil(t, fileInfo)
}

func assertIfFileNotExists(t *testing.T, fs afero.Fs, path string) {
	fileInfo, err := fs.Stat(path)
	assert.Error(t, err)
	assert.Nil(t, fileInfo)
}
