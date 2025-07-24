package bootstrapper

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapArgs(t *testing.T) {
	t.Run("check if all required flags are set", func(t *testing.T) {
		const (
			targetValue                   = "test-target-directory"
			versionValue                  = "test-version"
			suppressErrorValue            = false
			technologiesJava              = "java"
			technologiesGo                = "go"
			technologiesValue             = technologiesJava + "," + technologiesGo
			flavorValue                   = "flavor"
			inputDirectoryValue           = "test-input-directory"
			configDirectoryValue          = "test-config-directory"
			installPathValue              = "test-install-path"
			attributeContainerNameValue   = "container-name=test-container-name"
			attributeContainerLimitsValue = "container-limits=test-container-limits"
			attributeNamespaceValue       = "namespace-name=test-namespace"
			attributeWorkloadValue        = "statefulset-name=test-statefulset"
		)

		cmd := New()
		cmd.RunE = nil

		cmd.SetArgs([]string{
			"bootstrap",
			"--target=" + targetValue,
			"--version=" + versionValue,
			"--suppress-error=" + "false",
			"--technology=" + technologiesValue,
			"--flavor=" + flavorValue,
			"--input-directory=" + inputDirectoryValue,
			"--config-directory=" + configDirectoryValue,
			"--install-path=" + installPathValue,
			"--attribute-container=" + attributeContainerNameValue,
			"--attribute-container=" + attributeContainerLimitsValue,
			"--attribute=" + attributeNamespaceValue,
			"--attribute=" + attributeWorkloadValue,
		})

		err := cmd.Execute()
		require.NoError(t, err)

		value, err := cmd.Flags().GetString(TargetFolderFlag)
		require.NoError(t, err)
		assert.Equal(t, targetValue, value)

		value, err = cmd.Flags().GetString(TargetVersionFlag)
		require.NoError(t, err)
		assert.Equal(t, versionValue, value)

		suppress, err := cmd.Flags().GetBool(SuppressErrorsFlag)
		require.NoError(t, err)
		assert.Equal(t, suppressErrorValue, suppress)

		technologies, err = cmd.Flags().GetStringSlice(TechnologiesFlag)
		require.NoError(t, err)
		assert.Equal(t, technologiesJava, technologies[0])
		assert.Equal(t, technologiesGo, technologies[1])

		value, err = cmd.Flags().GetString(FlavorFlag)
		require.NoError(t, err)
		assert.Equal(t, flavorValue, value)

		value, err = cmd.Flags().GetString(configure.InputFolderFlag)
		require.NoError(t, err)
		assert.Equal(t, inputDirectoryValue, value)

		value, err = cmd.Flags().GetString(configure.ConfigFolderFlag)
		require.NoError(t, err)
		assert.Equal(t, configDirectoryValue, value)

		value, err = cmd.Flags().GetString(configure.InstallPathFlag)
		require.NoError(t, err)
		assert.Equal(t, installPathValue, value)

		attributeContainer, err := cmd.Flags().GetStringArray(container.Flag)
		require.NoError(t, err)
		assert.Equal(t, attributeContainerNameValue, attributeContainer[0])
		assert.Equal(t, attributeContainerLimitsValue, attributeContainer[1])

		attribute, err := cmd.Flags().GetStringArray(pod.Flag)
		require.NoError(t, err)
		assert.Equal(t, attributeNamespaceValue, attribute[0])
		assert.Equal(t, attributeWorkloadValue, attribute[1])
	})
}

func TestBootstrapConfigurationStep(t *testing.T) {
	const (
		targetPath = "/mnt/bin"
		configPath = "/mnt/config"
		inputPath  = "/mnt/input"
	)

	setupFS := func(t *testing.T) afero.Afero {
		t.Helper()
		fs := afero.Afero{Fs: afero.NewMemMapFs()}

		createFile(t, fs, filepath.Join(inputPath, "ruxitagentproc.json"), `
{
  "properties": [
    {
      "section": "general",
      "key": "hostGroup",
      "value": "test-host-group"
    }
  ],
  "revision": 123
}
`)

		createFile(t, fs, filepath.Join(targetPath, "agent/conf/ruxitagentproc.conf"), "[general]\n")

		createFile(t, fs, filepath.Join(inputPath, "endpoint.properties"), `DT_METRICS_INGEST_URL=http://ingest-url
	DT_METRICS_INGEST_API_TOKEN=apitoken`)

		return fs
	}

	t.Run("only OA", func(t *testing.T) {
		fs := setupFS(t)

		cmd := newCmd(fs)
		cmd.SetArgs([]string{
			"bootstrap",
			"--metadata-enrichment=false",
			"--config-directory=" + configPath,
			"--input-directory=" + inputPath,
			"--attribute-container={\"k8s.container.name\":\"test-container\"}",
			"--source=/opt/dynatrace/oneagent",
			"--target=" + targetPath,
			"--install-path=/opt/dynatrace/oneagent-paas",
			"--technologies=java",
			"--attribute=k8s.cluster.name=dynakube",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		exists, err := fs.Exists(filepath.Join(configPath, preload.ConfigPath))
		require.NoError(t, err)
		assert.True(t, exists)

		containerConfigPath := filepath.Join(configPath, "test-container")

		exists, err = fs.Exists(pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = fs.Exists(filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("only metadata", func(t *testing.T) {
		fs := setupFS(t)

		cmd := newCmd(fs)
		cmd.SetArgs([]string{
			"bootstrap",
			"--metadata-enrichment",
			"--config-directory=" + configPath,
			"--input-directory=" + inputPath,
			"--attribute-container={\"k8s.container.name\":\"test-container\"}",
			"--attribute=k8s.cluster.name=dynakube",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		exists, err := fs.Exists(filepath.Join(configPath, preload.ConfigPath))
		require.NoError(t, err)
		assert.False(t, exists)

		containerConfigPath := filepath.Join(configPath, "test-container")

		exists, err = fs.Exists(pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))
		require.NoError(t, err)
		assert.False(t, exists)

		exists, err = fs.Exists(filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("OA + metadata", func(t *testing.T) {
		fs := setupFS(t)

		cmd := newCmd(fs)
		cmd.SetArgs([]string{
			"bootstrap",
			"--metadata-enrichment",
			"--config-directory=" + configPath,
			"--input-directory=" + inputPath,
			"--attribute-container={\"k8s.container.name\":\"test-container\"}",
			"--source=/opt/dynatrace/oneagent",
			"--target=" + targetPath,
			"--install-path=/opt/dynatrace/oneagent-paas",
			"--technologies=java",
			"--attribute=k8s.cluster.name=dynakube",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		exists, err := fs.Exists(filepath.Join(configPath, preload.ConfigPath))
		require.NoError(t, err)
		assert.True(t, exists)

		containerConfigPath := filepath.Join(configPath, "test-container")

		exists, err = fs.Exists(pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = fs.Exists(filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func createFile(t *testing.T, fs afero.Fs, filePath string, content string) {
	file, err := fs.Create(filePath)
	require.NoError(t, err)

	if content != "" {
		_, err = file.Write([]byte(content))
		require.NoError(t, err)
	}

	file.Close()
}
