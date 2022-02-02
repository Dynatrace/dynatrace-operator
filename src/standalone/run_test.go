package standalone

import (
	"fmt"
	"path/filepath"
	"testing"

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
	t.Run(`create runner`, func(t *testing.T) {
		runner := creatTestRunner(t)
		assert.NotNil(t, runner.fs)
		assert.NotNil(t, runner.env)
		assert.NotNil(t, runner.dtclient)
		assert.NotNil(t, runner.config)
		assert.NotNil(t, runner.installer)
		assert.Empty(t, runner.hostTenant)
	})
}

func TestConsumeErrorIfNecessary(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run(`consume error`, func(t *testing.T) {
		runner.env.canFail = false
		err := fmt.Errorf("TESTING")

		runner.consumeErrorIfNecessary(&err)
		assert.NoError(t, err)
	})
	t.Run(`NOT consume error`, func(t *testing.T) {
		runner.env.canFail = true
		err := fmt.Errorf("TESTING")

		runner.consumeErrorIfNecessary(&err)
		assert.Error(t, err)
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
		runner.env.k8NodeName = testNodeName

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, testNodeIP, runner.hostTenant)
	})
	t.Run(`set hostTenant to empty`, func(t *testing.T) {
		runner.config.HasHost = false

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, noHostTenant, runner.hostTenant)
	})
}

func TestInstallOneAgent(t *testing.T) {
	t.Run(`happy install`, func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(nil)

		err := runner.installOneAgent()

		require.NoError(t, err)
	})
	t.Run(`sad install`, func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(fmt.Errorf("BOOM"))

		err := runner.installOneAgent()

		require.Error(t, err)
	})
}

func TestConfigureInstallation(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false
	runner.dtclient.(*dtclient.MockDynatraceClient).
		On("GetProcessModuleConfig", uint(0)).
		Return(&testProcessModuleConfig, nil)
	runner.installer.(*installer.InstallerMock).
		On("UpdateProcessModuleConfig", BinDirMount, &testProcessModuleConfig).
		Return(nil)

	t.Run(`create all config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.oneAgentInjected = true
		runner.env.dataIngestInjected = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		// container confs
		for _, container := range runner.env.containers {
			assertIfFileExists(t,
				runner.fs,
				filepath.Join(
					ShareDirMount,
					fmt.Sprintf(ContainerConfFilenameTemplate, container.name)))
		}
		// tls cert
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(ShareDirMount, "custom.pem"))
		// enrichemt
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "json")))
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "properties")))

	})
	t.Run(`create only container confs`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.oneAgentInjected = true
		runner.env.dataIngestInjected = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		// container confs
		for _, container := range runner.env.containers {
			assertIfFileExists(t,
				runner.fs,
				filepath.Join(
					ShareDirMount,
					fmt.Sprintf(ContainerConfFilenameTemplate, container.name)))
		}
		// tls cert
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(ShareDirMount, "custom.pem"))
		// enrichemt
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "json")))
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "properties")))

	})
	t.Run(`create only enrichment file`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.oneAgentInjected = false
		runner.env.dataIngestInjected = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		// container confs
		for _, container := range runner.env.containers {
			assertIfFileNotExists(t,
				runner.fs,
				filepath.Join(
					ShareDirMount,
					fmt.Sprintf(ContainerConfFilenameTemplate, container.name)))
		}
		// tls cert
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(ShareDirMount, "custom.pem"))
		// enrichemt
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "json")))
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "properties")))

	})
}

func TestCreateContainerConfigurationFiles(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run(`create config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.createContainerConfigurationFiles()

		require.NoError(t, err)
		for _, container := range runner.env.containers {
			assertIfFileExists(t,
				runner.fs,
				filepath.Join(
					ShareDirMount,
					fmt.Sprintf(ContainerConfFilenameTemplate, container.name)))
		}
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
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "json")))
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				EnrichmentPath,
				fmt.Sprintf(EnrichmentFilenameTemplate, "properties")))
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
			filepath.Join(ShareDirMount, "custom.pem"))
		// TODO: Check content ?
	})
}

func creatTestRunner(t *testing.T) *Runner {
	fs := prepTestFs(t)
	resetEnv := prepTestEnv(t)

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
