package startup

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestProcessModuleConfig() *dtclient.ProcessModuleConfig {
	return &dtclient.ProcessModuleConfig{
		Revision: 0,
		Properties: []dtclient.ProcessModuleProperty{
			{
				Section: "test",
				Key:     "test",
				Value:   "test",
			},
		},
	}
}

func TestNewRunner(t *testing.T) {
	fs := prepTestFs(t)
	t.Run("create runner with oneagent and data-ingest injection", func(t *testing.T) {
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
	t.Run("create runner with only oneagent", func(t *testing.T) {
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
	t.Run("create runner with only data-ingest injection", func(t *testing.T) {
		resetEnv := prepDataIngestTestEnv(t, false)
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
	t.Run("no error thrown", func(t *testing.T) {
		runner.env.FailurePolicy = silentPhrase
		err := runner.Run()
		assert.Nil(t, err)
	})
	t.Run("error thrown, but consume error", func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.FailurePolicy = silentPhrase
		err := runner.Run()
		assert.Nil(t, err)
	})
	t.Run("error thrown, but don't consume error", func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.FailurePolicy = failPhrase
		err := runner.Run()
		assert.NotNil(t, err)
	})
}

func TestSetHostTenant(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run("fail due to missing node", func(t *testing.T) {
		runner.config.HasHost = true

		err := runner.setHostTenant()

		require.Error(t, err)
	})
	t.Run("set hostTenant to node", func(t *testing.T) {
		runner.config.HasHost = true
		runner.env.K8NodeName = testNodeName

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, runner.hostTenant)
	})
	t.Run("set hostTenant to empty", func(t *testing.T) {
		runner.config.HasHost = false

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, consts.AgentNoHostTenant, runner.hostTenant)
	})
	t.Run("set hostTenant to TenantUUID", func(t *testing.T) {
		runner.env.FailurePolicy = forcePhrase
		runner.config.HasHost = true
		runner.config.TenantUUID = testTenantUUID

		err := runner.setHostTenant()

		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, runner.hostTenant)
	})
}

func TestInstallOneAgent(t *testing.T) {
	t.Run("happy install", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.fs.Create(filepath.Join(consts.AgentBinDirMount, "agent/conf/ruxitagentproc.conf"))
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(getTestProcessModuleConfig(), nil)
		runner.installer.(*installer.Mock).
			On("InstallAgent", consts.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.NoError(t, err)
	})
	t.Run("sad install -> install fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.installer.(*installer.Mock).
			On("InstallAgent", consts.AgentBinDirMount).
			Return(false, fmt.Errorf("BOOM"))

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run("sad install -> ruxitagent update fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(getTestProcessModuleConfig(), nil)
		runner.installer.(*installer.Mock).
			On("InstallAgent", consts.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent()

		require.Error(t, err)
	})
	t.Run("sad install -> ruxitagent endpoint fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))
		runner.installer.(*installer.Mock).
			On("InstallAgent", consts.AgentBinDirMount).
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
		Return(getTestProcessModuleConfig(), nil)

	t.Run("no install, just config generation", func(t *testing.T) {
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.env.Mode = consts.AgentCsiMode

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("install + config generation", func(t *testing.T) {
		runner.installer.(*installer.Mock).
			On("InstallAgent", consts.AgentBinDirMount).
			Return(true, nil)
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.env.Mode = consts.AgentInstallerMode
		runner.fs.Create(filepath.Join(consts.AgentBinDirMount, "agent/conf/ruxitagentproc.conf"))

		err := runner.Run()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
}

func TestConfigureInstallation(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run("create all config files", func(t *testing.T) {
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.env.OneAgentInjected = true
		runner.env.DataIngestInjected = true
		runner.env.IsReadOnlyCSI = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("create only container confs", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = true
		runner.env.DataIngestInjected = false
		runner.env.IsReadOnlyCSI = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesNotExists(t, *runner)
	})
	t.Run("create only container confs with readonly csi", func(t *testing.T) {
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.env.OneAgentInjected = true
		runner.env.DataIngestInjected = false
		runner.env.IsReadOnlyCSI = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesNotExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("create only enrichment file", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = false
		runner.env.DataIngestInjected = true
		runner.env.IsReadOnlyCSI = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesNotExists(t, *runner)
		// enrichemt
		assertIfEnrichmentFilesExists(t, *runner)
	})
}

func TestGetProcessModuleConfig(t *testing.T) {
	t.Run("error if api call fails", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))

		config, err := runner.getProcessModuleConfig()
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("add proxy to process module config", func(t *testing.T) {
		const proxy = "dummy-proxy"
		runner := createMockedRunner(t)
		runner.config.Proxy = proxy
		runner.dtclient.(*dtclient.MockDynatraceClient).
			On("GetProcessModuleConfig", uint(0)).
			Return(getTestProcessModuleConfig(), nil)

		config, err := runner.getProcessModuleConfig()
		require.NoError(t, err)
		require.NotNil(t, config)

		generalSection, ok := config.ToMap()["general"]
		require.True(t, ok)
		value, ok := generalSection["proxy"]
		require.True(t, ok)
		assert.Equal(t, proxy, value)
	})
}

func TestCreateContainerConfigurationsFiles(t *testing.T) {
	const expectedContainerConfContentAppMon = `[container]
containerName TEST_CONTAINER_%d_NAME
imageName TEST_CONTAINER_%d_IMAGE
k8s_fullpodname TEST_K8S_PODNAME
k8s_poduid TEST_K8S_PODUID
k8s_containername TEST_CONTAINER_%d_NAME
k8s_basepodname TEST_K8S_BASEPODNAME
k8s_namespace TEST_K8S_NAMESPACE
k8s_cluster_id TEST_K8S_CLUSTER_ID
`

	const expectedContainerConfContentCloudNative = expectedContainerConfContentAppMon + `k8s_node_name TEST_K8S_NODE_NAME
[host]
tenant test
isCloudNativeFullStack true
`
	runner := createMockedRunner(t)

	t.Run("create config files in case of application monitoring", func(t *testing.T) {
		runner.config.HasHost = false
		runner.fs = afero.NewMemMapFs()

		err := runner.createContainerConfigurationFiles()

		require.NoError(t, err)
		for i, container := range runner.env.Containers {
			filePath := filepath.Join(
				consts.AgentShareDirMount,
				fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, container.Name))

			assertIfFileExists(t, runner.fs, filePath)

			file, err := runner.fs.Open(filePath)
			require.NoError(t, err)

			info, err := file.Stat()
			require.NoError(t, err)

			content := make([]byte, info.Size())
			n, err := file.Read(content)

			require.Equal(t, info.Size(), int64(n))
			require.NoError(t, err)

			assert.Equal(t, fmt.Sprintf(expectedContainerConfContentAppMon, i+1, i+1, i+1), string(content))
		}
	})
	t.Run("create config files in case of cloud native fullstack", func(t *testing.T) {
		runner.config.HasHost = true
		runner.hostTenant = testTenantUUID
		runner.fs = afero.NewMemMapFs()

		err := runner.createContainerConfigurationFiles()

		require.NoError(t, err)
		for i, container := range runner.env.Containers {
			filePath := filepath.Join(
				consts.AgentShareDirMount,
				fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, container.Name))

			assertIfFileExists(t, runner.fs, filePath)

			file, err := runner.fs.Open(filePath)
			require.NoError(t, err)

			info, err := file.Stat()
			require.NoError(t, err)

			content := make([]byte, info.Size())
			n, err := file.Read(content)

			require.Equal(t, info.Size(), int64(n))
			require.NoError(t, err)

			assert.Equal(t, fmt.Sprintf(expectedContainerConfContentCloudNative, i+1, i+1, i+1), string(content))
		}
	})
}

func TestSetLDPreload(t *testing.T) {
	runner := createMockedRunner(t)
	t.Run("create ld preload file", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.setLDPreload()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.AgentShareDirMount,
				consts.LdPreloadFilename))
		// TODO: Check content ?
	})
}

func TestEnrichMetadata(t *testing.T) {
	runner := createMockedRunner(t)
	runner.config.HasHost = false

	t.Run("create enrichment files", func(t *testing.T) {
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

	t.Run("create tls custom.pem", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()

		err := runner.propagateTLSCert()

		require.NoError(t, err)
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(consts.AgentShareDirMount, "custom.pem"))
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

func createTestRunner(t *testing.T) *Runner {
	fs := prepTestFs(t)
	resetEnv := prepCombinedTestEnv(t)

	runner, err := NewRunner(fs)
	resetEnv()
	require.NoError(t, err)
	require.NotNil(t, runner)
	return runner
}

func createMockedRunner(t *testing.T) *Runner {
	runner := createTestRunner(t)
	runner.installer = &installer.Mock{}
	runner.dtclient = &dtclient.MockDynatraceClient{}
	return runner
}

func assertIfAgentFilesExists(t *testing.T, runner Runner) {
	// container confs
	for _, container := range runner.env.Containers {
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.AgentShareDirMount,
				fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(consts.AgentShareDirMount, consts.LdPreloadFilename))
	// tls cert
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(consts.AgentShareDirMount, "custom.pem"))
}

func assertIfEnrichmentFilesExists(t *testing.T, runner Runner) {
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(
			consts.EnrichmentMountPath,
			fmt.Sprintf(consts.EnrichmentFilenameTemplate, "json")))
	assertIfFileExists(t,
		runner.fs,
		filepath.Join(
			consts.EnrichmentMountPath,
			fmt.Sprintf(consts.EnrichmentFilenameTemplate, "properties")))
}

func assertIfAgentFilesNotExists(t *testing.T, runner Runner) {
	// container confs
	for _, container := range runner.env.Containers {
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				consts.AgentShareDirMount,
				fmt.Sprintf(consts.AgentContainerConfFilenameTemplate, container.Name)))
	}
	// ld.so.preload
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(consts.AgentShareDirMount, consts.LdPreloadFilename))
	// tls cert
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(consts.AgentShareDirMount, "custom.pem"))
}

func assertIfEnrichmentFilesNotExists(t *testing.T, runner Runner) {
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(
			consts.EnrichmentMountPath,
			fmt.Sprintf(consts.EnrichmentFilenameTemplate, "json")))
	assertIfFileNotExists(t,
		runner.fs,
		filepath.Join(
			consts.EnrichmentMountPath,
			fmt.Sprintf(consts.EnrichmentFilenameTemplate, "properties")))
}

func assertIfReadOnlyCSIFilesExists(t *testing.T, runner Runner) {
	for i := 0; i < 10; i++ {
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.AgentConfInitDirMount,
				fmt.Sprintf("%d.conf", i)))
	}
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
