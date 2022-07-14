package standalone

import (
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnv(t *testing.T) {
	t.Run(`create new env for oneagent and data-ingest injection`, func(t *testing.T) {
		resetEnv := prepCombinedTestEnv(t)

		env, err := newEnv()
		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, config.AgentCsiMode, env.Mode)
		assert.True(t, env.CanFail)
		assert.NotEmpty(t, env.InstallerFlavor)
		assert.NotEmpty(t, env.InstallerTech)
		assert.NotEmpty(t, env.InstallPath)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8Namespace)
		assert.NotEmpty(t, env.K8ClusterID)

		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)

		assert.True(t, env.OneAgentInjected)
		assert.True(t, env.DataIngestInjected)
	})
	t.Run(`create new env for only data-ingest injection`, func(t *testing.T) {
		resetEnv := prepDataIngestTestEnv(t)

		env, err := newEnv()
		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Empty(t, env.Mode)
		assert.True(t, env.CanFail)
		assert.NotEmpty(t, env.InstallerFlavor) // set to what is defined in arch.Flavor
		assert.Empty(t, env.InstallerTech)
		assert.Empty(t, env.InstallPath)
		assert.Empty(t, env.Containers)

		assert.Empty(t, env.K8NodeName)
		assert.Empty(t, env.K8PodName)
		assert.Empty(t, env.K8PodUID)
		assert.Empty(t, env.K8BasePodName)
		assert.Empty(t, env.K8Namespace)

		assert.NotEmpty(t, env.K8ClusterID)
		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)

		assert.False(t, env.OneAgentInjected)
		assert.True(t, env.DataIngestInjected)
	})
	t.Run(`create new env for only oneagent`, func(t *testing.T) {
		resetEnv := prepOneAgentTestEnv(t)

		env, err := newEnv()
		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, config.AgentCsiMode, env.Mode)
		assert.True(t, env.CanFail)
		assert.NotEmpty(t, env.InstallerFlavor)
		assert.NotEmpty(t, env.InstallerTech)
		assert.NotEmpty(t, env.InstallPath)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8Namespace)

		assert.Empty(t, env.K8ClusterID)
		assert.Empty(t, env.WorkloadKind)
		assert.Empty(t, env.WorkloadName)

		assert.True(t, env.OneAgentInjected)
		assert.False(t, env.DataIngestInjected)
	})
}

func prepCombinedTestEnv(t *testing.T) func() {
	resetDataIngestEnvs := prepDataIngestTestEnv(t)
	resetOneAgentEnvs := prepOneAgentTestEnv(t)
	return func() {
		resetDataIngestEnvs()
		resetOneAgentEnvs()
	}
}

func prepOneAgentTestEnv(t *testing.T) func() {
	envs := []string{
		config.AgentInstallerFlavorEnv,
		config.AgentInstallerTechEnv,
		config.K8NodeNameEnv,
		config.K8PodNameEnv,
		config.K8PodUIDEnv,
		config.K8BasePodNameEnv,
		config.K8NamespaceEnv,
		config.AgentInstallPathEnv,
	}
	for i := 1; i <= 5; i++ {
		envs = append(envs, fmt.Sprintf(config.AgentContainerNameEnvTemplate, i))
		envs = append(envs, fmt.Sprintf(config.AgentContainerImageEnvTemplate, i))
	}
	for _, envvar := range envs {
		err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		require.NoError(t, err)
	}

	// Int env
	envs = append(envs, config.AgentContainerCountEnv)
	err := os.Setenv(config.AgentContainerCountEnv, "5")
	require.NoError(t, err)

	// Mode Env
	envs = append(envs, config.InjectionCanFailEnv)
	err = os.Setenv(config.InjectionCanFailEnv, "fail")
	require.NoError(t, err)
	envs = append(envs, config.AgentInstallModeEnv)
	err = os.Setenv(config.AgentInstallModeEnv, string(config.AgentCsiMode))
	require.NoError(t, err)

	// Bool envs
	err = os.Setenv(config.AgentInjectedEnv, "true")
	require.NoError(t, err)
	envs = append(envs, config.AgentInjectedEnv)

	return resetTestEnv(envs)
}

func prepDataIngestTestEnv(t *testing.T) func() {
	envs := []string{
		config.EnrichmentWorkloadKindEnv,
		config.EnrichmentWorkloadNameEnv,
		config.K8ClusterIDEnv,
	}
	for _, envvar := range envs {
		err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		require.NoError(t, err)
	}

	// Mode Env
	envs = append(envs, config.InjectionCanFailEnv)
	err := os.Setenv(config.InjectionCanFailEnv, "fail")
	require.NoError(t, err)

	// Bool envs
	err = os.Setenv(config.EnrichmentInjectedEnv, "true")
	require.NoError(t, err)
	envs = append(envs, config.EnrichmentInjectedEnv)

	return resetTestEnv(envs)
}

func resetTestEnv(envs []string) func() {
	return func() {
		for _, envvar := range envs {
			os.Unsetenv(envvar)
		}
	}
}
