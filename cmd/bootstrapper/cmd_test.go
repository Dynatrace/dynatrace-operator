package bootstrapper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
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
	setupFS := func(t *testing.T, inputPath, targetPath string) {
		t.Helper()

		createFile(t, filepath.Join(inputPath, "ruxitagentproc.json"), `
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

		createFile(t, filepath.Join(targetPath, "agent/conf/ruxitagentproc.conf"), "[general]\n")

		createFile(t, filepath.Join(inputPath, "endpoint.properties"), `DT_METRICS_INGEST_URL=http://ingest-url
	DT_METRICS_INGEST_API_TOKEN=apitoken`)
	}

	t.Run("only OA", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "/mnt", "bin")
		configPath := filepath.Join(tmpDir, "/mnt", "config")
		inputPath := filepath.Join(tmpDir, "/mnt", "input")
		setupFS(t, inputPath, targetPath)

		cmd := newCmd()
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

		assert.FileExists(t, filepath.Join(configPath, preload.ConfigPath))

		containerConfigPath := filepath.Join(configPath, "test-container")
		assert.FileExists(t, pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))

		assert.NoFileExists(t, filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
	})

	t.Run("only metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "/mnt", "bin")
		configPath := filepath.Join(tmpDir, "/mnt", "config")
		inputPath := filepath.Join(tmpDir, "/mnt", "input")
		setupFS(t, inputPath, targetPath)

		cmd := newCmd()
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

		assert.NoFileExists(t, filepath.Join(configPath, preload.ConfigPath))

		containerConfigPath := filepath.Join(configPath, "test-container")
		assert.NoFileExists(t, pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))

		assert.FileExists(t, filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
	})

	t.Run("OA + metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "/mnt", "bin")
		configPath := filepath.Join(tmpDir, "/mnt", "config")
		inputPath := filepath.Join(tmpDir, "/mnt", "input")
		setupFS(t, inputPath, targetPath)

		cmd := newCmd()
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

		assert.FileExists(t, filepath.Join(configPath, preload.ConfigPath))

		containerConfigPath := filepath.Join(configPath, "test-container")
		assert.FileExists(t, pmc.GetDestinationRuxitAgentProcFilePath(containerConfigPath))

		assert.FileExists(t, filepath.Join(containerConfigPath, "enrichment/endpoint/endpoint.properties"))
	})
}

func createFile(t *testing.T, filePath string, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), os.ModePerm))

	file, err := os.Create(filePath)
	require.NoError(t, err)

	if content != "" {
		_, err = file.WriteString(content)
		require.NoError(t, err)
	}

	file.Close()
}
