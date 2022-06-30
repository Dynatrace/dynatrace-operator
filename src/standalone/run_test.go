package standalone

import (
	"io/ioutil"
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
		assert.NotNil(t, runner.agentSetup)
		assert.NotNil(t, runner.ingestSetup)
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

func TestRun(t *testing.T) {
	runner := createMockedRunner(t)
	runner.agentSetup.config.HasHost = false
	runner.env.OneAgentInjected = true
	runner.env.DataIngestInjected = true
	runner.agentSetup.dtclient.(*dtclient.MockDynatraceClient).
		On("GetProcessModuleConfig", uint(0)).
		Return(&testProcessModuleConfig, nil)
	runner.agentSetup.installer.(*installer.InstallerMock).
		On("UpdateProcessModuleConfig", BinDirMount, &testProcessModuleConfig).
		Return(nil)

	t.Run(`no install, just config generation`, func(t *testing.T) {
		resetRunnerTestFs(runner)
		runner.env.Mode = CsiMode

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner.agentSetup)
		assertIfEnrichmentFilesExists(t, *runner.ingestSetup)

	})
	t.Run(`install + config generation`, func(t *testing.T) {
		runner.agentSetup.installer.(*installer.InstallerMock).
			On("InstallAgent", BinDirMount).
			Return(true, nil)
		resetRunnerTestFs(runner)
		runner.env.Mode = InstallerMode

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner.agentSetup)
		assertIfEnrichmentFilesExists(t, *runner.ingestSetup)

	})
}

func TestCreateConfFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run(`create file`, func(t *testing.T) {
		path := "test"

		err := createConfFile(fs, path, "test")

		require.NoError(t, err)

		file, err := fs.Open(path)
		require.NoError(t, err)
		content, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test")

	})
	t.Run(`create nested file`, func(t *testing.T) {
		path := filepath.Join("dir1", "dir2", "test")

		err := createConfFile(fs, path, "test")

		require.NoError(t, err)

		file, err := fs.Open(path)
		require.NoError(t, err)
		content, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test")

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

func resetRunnerTestFs(runner *Runner) {
	fs := afero.NewMemMapFs()
	runner.fs = fs
	runner.ingestSetup.fs = fs
	runner.agentSetup.fs = fs
}

func createMockedRunner(t *testing.T) *Runner {
	runner := creatTestRunner(t)
	ingestSetup := createTestDataIngestSetup(t)
	agentSetup := createMockedOneAgentSetup(t)
	ingestSetup.fs = runner.fs
	agentSetup.fs = runner.fs
	runner.ingestSetup = ingestSetup
	runner.agentSetup = agentSetup
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
