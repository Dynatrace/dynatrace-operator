package startup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	installermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/injection/codemodule/installer"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	t.Run("create runner with oneagent and metadata-enrichment injection", func(t *testing.T) {
		prepCombinedTestEnv(t)

		runner, err := NewRunner(fs)

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
		prepOneAgentTestEnv(t)

		runner, err := NewRunner(fs)

		require.NoError(t, err)
		assert.NotNil(t, runner.fs)
		assert.NotNil(t, runner.env)
		assert.NotNil(t, runner.dtclient)
		assert.NotNil(t, runner.config)
		assert.NotNil(t, runner.installer)
		assert.Empty(t, runner.hostTenant)
	})
	t.Run("create runner with only metadata-enrichment injection", func(t *testing.T) {
		prepMetadataEnrichmentTestEnv(t, false)

		runner, err := NewRunner(fs)

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
	ctx := context.Background()
	runner := createMockedRunner(t)
	t.Run("no error thrown", func(t *testing.T) {
		runner.env.FailurePolicy = silentPhrase
		err := runner.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("error thrown, but consume error", func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.FailurePolicy = silentPhrase
		err := runner.Run(ctx)
		require.NoError(t, err)
	})
	t.Run("error thrown, but don't consume error", func(t *testing.T) {
		runner.env.K8NodeName = "" // create artificial error
		runner.env.FailurePolicy = failPhrase
		err := runner.Run(ctx)
		require.Error(t, err)
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
		runner.config.HasHost = true
		runner.config.TenantUUID = testTenantUUID
		runner.config.EnforcementMode = false
		runner.config.MonitoringNodes = nil

		err := runner.setHostTenant()

		require.Error(t, err)
	})
}

func TestInstallOneAgent(t *testing.T) {
	ctx := context.Background()

	t.Run("happy install", func(t *testing.T) {
		runner := createMockedRunner(t)
		_, err := runner.fs.Create(filepath.Join(consts.AgentBinDirMount, "agent/conf/ruxitagentproc.conf"))
		require.NoError(t, err)
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(getTestProcessModuleConfig(), nil)
		runner.installer.(*installermock.Installer).
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), consts.AgentBinDirMount).
			Return(true, nil)

		err = runner.installOneAgent(ctx)

		require.NoError(t, err)
	})
	t.Run("sad install -> install fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.installer.(*installermock.Installer).
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), consts.AgentBinDirMount).
			Return(false, fmt.Errorf("BOOM"))

		err := runner.installOneAgent(ctx)

		require.Error(t, err)
	})
	t.Run("sad install -> ruxitagent update fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(getTestProcessModuleConfig(), nil)
		runner.installer.(*installermock.Installer).
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), consts.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent(ctx)

		require.Error(t, err)
	})
	t.Run("sad install -> ruxitagent endpoint fail", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))
		runner.installer.(*installermock.Installer).
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), consts.AgentBinDirMount).
			Return(true, nil)

		err := runner.installOneAgent(ctx)

		require.Error(t, err)
	})
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	runner := createMockedRunner(t)
	runner.config.HasHost = false
	runner.env.OneAgentInjected = true
	runner.env.MetadataEnrichmentInjected = true
	runner.dtclient.(*dtclientmock.Client).
		On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
		Return(getTestProcessModuleConfig(), nil)

	t.Run("no install, just config generation", func(t *testing.T) {
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.config.CSIMode = true
		runner.config.ReadOnlyCSIDriver = true

		err := runner.Run(ctx)

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("install + config generation", func(t *testing.T) {
		runner.installer.(*installermock.Installer).
			On("InstallAgent", mock.AnythingOfType("context.backgroundCtx"), consts.AgentBinDirMount).
			Return(true, nil)

		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.config.CSIMode = false
		_, err := runner.fs.Create(filepath.Join(consts.AgentBinDirMount, "agent/conf/ruxitagentproc.conf"))
		require.NoError(t, err)

		err = runner.Run(ctx)

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
		runner.env.MetadataEnrichmentInjected = true
		runner.config.ReadOnlyCSIDriver = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("create only container confs", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = true
		runner.env.MetadataEnrichmentInjected = false
		runner.config.ReadOnlyCSIDriver = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesNotExists(t, *runner)
	})
	t.Run("create only container confs with readonly csi", func(t *testing.T) {
		runner.fs = prepReadOnlyCSIFilesystem(t, afero.NewMemMapFs())
		runner.env.OneAgentInjected = true
		runner.env.MetadataEnrichmentInjected = false
		runner.config.ReadOnlyCSIDriver = true

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesExists(t, *runner)
		assertIfEnrichmentFilesNotExists(t, *runner)
		assertIfReadOnlyCSIFilesExists(t, *runner)
	})
	t.Run("create only enrichment file", func(t *testing.T) {
		runner.fs = afero.NewMemMapFs()
		runner.env.OneAgentInjected = false
		runner.env.MetadataEnrichmentInjected = true
		runner.config.ReadOnlyCSIDriver = false

		err := runner.configureInstallation()

		require.NoError(t, err)
		assertIfAgentFilesNotExists(t, *runner)
		// enrichemt
		assertIfEnrichmentFilesExists(t, *runner)
	})
}

func TestGetProcessModuleConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("error if api call fails", func(t *testing.T) {
		runner := createMockedRunner(t)
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(&dtclient.ProcessModuleConfig{}, fmt.Errorf("BOOM"))

		config, err := runner.getProcessModuleConfig(ctx)
		require.Error(t, err)
		require.Nil(t, config)
	})

	t.Run("add proxy to process module config", func(t *testing.T) {
		const proxy = "dummy-proxy"

		runner := createMockedRunner(t)
		runner.config.Proxy = proxy
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(getTestProcessModuleConfig(), nil)

		config, err := runner.getProcessModuleConfig(ctx)
		require.NoError(t, err)
		require.NotNil(t, config)

		generalSection, ok := config.ToMap()["general"]
		require.True(t, ok)
		value, ok := generalSection["proxy"]
		require.True(t, ok)
		assert.Equal(t, proxy, value)
	})

	t.Run("add proxy to process module config", func(t *testing.T) {
		const oneAgentNoProxy = "dummy-no-proxy"

		runner := createMockedRunner(t)
		runner.config.OneAgentNoProxy = oneAgentNoProxy
		runner.dtclient.(*dtclientmock.Client).
			On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), uint(0)).
			Return(getTestProcessModuleConfig(), nil)

		config, err := runner.getProcessModuleConfig(ctx)
		require.NoError(t, err)
		require.NotNil(t, config)

		generalSection, ok := config.ToMap()["general"]
		require.True(t, ok)
		value, ok := generalSection["noProxy"]
		require.True(t, ok)
		assert.Equal(t, oneAgentNoProxy, value)
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
	})
}

func TestPropagateTLSCert(t *testing.T) {
	prepOneAgentTestEnv(t)

	initSecretConfig := getTestSecretConfig()
	initSecretData, err := json.Marshal(initSecretConfig)
	require.NoError(t, err)

	t.Run("propagate combined certificates to custom.pem and custom_proxy.pem", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.AgentInitSecretConfigField, string(initSecretData))
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.ActiveGateCAsInitSecretField, "not empty")
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.TrustedCAsInitSecretField, "not empty")
		runner, err := NewRunner(fs)
		require.NoError(t, err)
		require.NotNil(t, runner)

		err = runner.propagateTLSCert()
		require.NoError(t, err)

		assertIfFileExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomCertsFileName))
		assertIfFileExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomProxyCertsFileName))
	})
	t.Run("create custom.pem only containing AG cert", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.AgentInitSecretConfigField, string(initSecretData))
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.ActiveGateCAsInitSecretField, "not empty")
		runner, err := NewRunner(fs)
		require.NoError(t, err)
		require.NotNil(t, runner)

		err = runner.propagateTLSCert()
		require.NoError(t, err)

		assertIfFileExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomCertsFileName))
		assertIfFileNotExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomProxyCertsFileName))
	})
	t.Run("propagate trustedCAs certificates to custom.pem and custom_proxy.pem", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.AgentInitSecretConfigField, string(initSecretData))
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.TrustedCAsInitSecretField, "not empty")
		runner, err := NewRunner(fs)
		require.NoError(t, err)
		require.NotNil(t, runner)

		err = runner.propagateTLSCert()
		require.NoError(t, err)

		assertIfFileExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomCertsFileName))
		assertIfFileExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomProxyCertsFileName))
	})
	t.Run("don't create cert files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.AgentInitSecretConfigField, string(initSecretData))
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.ActiveGateCAsInitSecretField, "")
		fs = prepTestFile(t, fs, consts.AgentConfigDirMount, consts.TrustedCAsInitSecretField, "")
		runner, err := NewRunner(fs)
		require.NoError(t, err)
		require.NotNil(t, runner)

		err = runner.propagateTLSCert()
		require.NoError(t, err)

		assertIfFileNotExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomCertsFileName))
		assertIfFileNotExists(t, runner.fs, filepath.Join(consts.AgentShareDirMount, consts.CustomProxyCertsFileName))
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

	require.NoError(t, err)

	exists, err := afero.Exists(filesystem, "/mnt/share/curl_options.conf")

	require.NoError(t, err)
	assert.True(t, exists)
}

func createTestRunner(t *testing.T) *Runner {
	fs := prepTestFs(t)
	prepCombinedTestEnv(t)

	runner, err := NewRunner(fs)

	require.NoError(t, err)
	require.NotNil(t, runner)

	return runner
}

func createMockedRunner(t *testing.T) *Runner {
	runner := createTestRunner(t)
	runner.installer = installermock.NewInstaller(t)
	runner.dtclient = dtclientmock.NewClient(t)

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
}

func assertIfEnrichmentFilesExists(t *testing.T, runner Runner) {
	for _, container := range runner.env.Containers {
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.EnrichmentInitPath,
				fmt.Sprintf(consts.EnrichmentInitJsonFilenameTemplate, container.Name)))
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.EnrichmentInitPath,
				fmt.Sprintf(consts.EnrichmentInitPropertiesFilenameTemplate, container.Name)))
	}
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
	for _, container := range runner.env.Containers {
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				consts.EnrichmentMountPath,
				fmt.Sprintf(consts.EnrichmentInitJsonFilenameTemplate, container.Name)))
		assertIfFileNotExists(t,
			runner.fs,
			filepath.Join(
				consts.EnrichmentMountPath,
				fmt.Sprintf(consts.EnrichmentInitPropertiesFilenameTemplate, container.Name)))
	}
}

func assertIfReadOnlyCSIFilesExists(t *testing.T, runner Runner) {
	for i := range 10 {
		assertIfFileExists(t,
			runner.fs,
			filepath.Join(
				consts.AgentConfInitDirMount,
				fmt.Sprintf("%d.conf", i)))
	}
}

func assertIfFileExists(t *testing.T, fs afero.Fs, path string) {
	fileInfo, err := fs.Stat(path)
	require.NoError(t, err)
	assert.NotNil(t, fileInfo)
}

func assertIfFileNotExists(t *testing.T, fs afero.Fs, path string) {
	fileInfo, err := fs.Stat(path)
	require.Error(t, err)
	assert.Nil(t, fileInfo)
}

func prepTestFile(t *testing.T, fs afero.Fs, dirname string, filename string, content string) afero.Fs {
	require.NoError(t, fs.MkdirAll(dirname, 0770))

	file, err := fs.OpenFile(filepath.Join(dirname, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0770)
	require.NoError(t, err)
	require.NotNil(t, file)

	_, err = file.Write([]byte(content))
	require.NoError(t, err)

	err = file.Close()
	require.NoError(t, err)

	return fs
}
