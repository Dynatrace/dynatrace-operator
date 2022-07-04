package standalone

import (
	"fmt"
	"os"
	"testing"

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

		assert.Equal(t, CsiMode, env.Mode)
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

		assert.Equal(t, CsiMode, env.Mode)
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
		InstallerFlavorEnv,
		InstallerTechEnv,
		K8NodeNameEnv,
		K8PodNameEnv,
		K8PodUIDEnv,
		K8BasePodNameEnv,
		K8NamespaceEnv,
		InstallPathEnv,
	}
	for i := 1; i <= 5; i++ {
		envs = append(envs, fmt.Sprintf(ContainerNameEnvTemplate, i))
		envs = append(envs, fmt.Sprintf(ContainerImageEnvTemplate, i))
	}
	for _, envvar := range envs {
		err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		require.NoError(t, err)
	}

	// Int env
	envs = append(envs, ContainerCountEnv)
	err := os.Setenv(ContainerCountEnv, "5")
	require.NoError(t, err)

	// Mode Env
	envs = append(envs, CanFailEnv)
	err = os.Setenv(CanFailEnv, "fail")
	require.NoError(t, err)
	envs = append(envs, ModeEnv)
	err = os.Setenv(ModeEnv, string(CsiMode))
	require.NoError(t, err)

	// Bool envs
	err = os.Setenv(OneAgentInjectedEnv, "true")
	require.NoError(t, err)
	envs = append(envs, OneAgentInjectedEnv)

	return resetTestEnv(envs)
}

func prepDataIngestTestEnv(t *testing.T) func() {
	envs := []string{
		WorkloadKindEnv,
		WorkloadNameEnv,
		K8ClusterIDEnv,
	}
	for _, envvar := range envs {
		err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		require.NoError(t, err)
	}

	// Mode Env
	envs = append(envs, CanFailEnv)
	err := os.Setenv(CanFailEnv, "fail")
	require.NoError(t, err)

	// Bool envs
	err = os.Setenv(DataIngestInjectedEnv, "true")
	require.NoError(t, err)
	envs = append(envs, DataIngestInjectedEnv)

	return resetTestEnv(envs)
}

func resetTestEnv(envs []string) func() {
	return func() {
		for _, envvar := range envs {
			os.Unsetenv(envvar)
		}
	}
}
