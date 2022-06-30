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

func TestSetHostTenant(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
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
		assert.Equal(t, NoHostTenant, runner.hostTenant)
	})
}

func TestInstallOneAgent(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
	t.Run(`happy install`, func(t *testing.T) {
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", BinDirMount, &testProcessModuleConfig).
			Return(nil)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.NoError(t, err)
	})
	t.Run(`sad install -> install fail`, func(t *testing.T) {
		runner := createMockedOneAgentSetup(t)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(false, fmt.Errorf("BOOM"))

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run(`sad install -> ruxitagent update fail`, func(t *testing.T) {
		runner := createMockedOneAgentSetup(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&testProcessModuleConfig, nil)
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", BinDirMount, &testProcessModuleConfig).
			Return(fmt.Errorf("BOOM"))
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run(`sad install -> ruxitagent endpoint fail`, func(t *testing.T) {
		runner := createMockedOneAgentSetup(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))
		runner.installer.(*installer.InstallerMock).
			On("UpdateProcessModuleConfig", BinDirMount, &testProcessModuleConfig).
			Return(nil)
		runner.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.Error(t, err)
	})
}

func TestConfigureInstallation(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
	runner.config.HasHost = false

	t.Run(`create all config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		assertIfAgentFilesNotExists(t, *runner)

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)

	})
}

func TestCreateContainerConfigurationFiles(t *testing.T) {
	runner := createMockedOneAgentSetup(t)
	runner.config.HasHost = false

	t.Run(`create config files`, func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.createContainerConfigurationFiles()

		require.NoError(t, err)
		for _, container := range runner.env.Containers {
			assertIfFileExists(t,
				runner.fs,
				filepath.Join(
					ShareDirMount,
					fmt.Sprintf(ContainerConfFilenameTemplate, container.Name)))
		}
	})
}

func createTestOneAgentSetup(t *testing.T) *oneAgentSetup {
	fs := prepTestFs(t)
	resetEnv := prepTestEnv(t)

	env, err := newEnv()
	require.NoError(t, err)

	setup, err := newOneagentSetup(fs, env)
	resetEnv()
	require.NoError(t, err)
	require.NotNil(t, setup)
	return setup
}

func createMockedOneAgentSetup(t *testing.T) *oneAgentSetup {
	setup := createTestOneAgentSetup(t)
	setup.installer = &installer.InstallerMock{}
	setup.dtclient = &dtclient.MockDynatraceClient{}
	return setup
}

func assertIfAgentFilesExists(t *testing.T, setup oneAgentSetup) {
	// container confs
	for _, container := range setup.env.Containers {
		assertIfFileExists(t,
			setup.fs,
			filepath.Join(
				ShareDirMount,
				fmt.Sprintf(ContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileExists(t,
		setup.fs,
		filepath.Join(ShareDirMount, ldPreloadFilename))
	// tls cert
	assertIfFileExists(t,
		setup.fs,
		filepath.Join(ShareDirMount, "custom.pem"))

}

func assertIfAgentFilesNotExists(t *testing.T, setup oneAgentSetup) {
	// container confs
	for _, container := range setup.env.Containers {
		assertIfFileNotExists(t,
			setup.fs,
			filepath.Join(
				ShareDirMount,
				fmt.Sprintf(ContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileNotExists(t,
		setup.fs,
		filepath.Join(ShareDirMount, ldPreloadFilename))
	// tls cert
	assertIfFileNotExists(t,
		setup.fs,
		filepath.Join(ShareDirMount, "custom.pem"))

}
