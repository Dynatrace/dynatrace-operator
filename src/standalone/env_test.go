package standalone

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnv(t *testing.T) {
	t.Run(`create new env`, func(t *testing.T) {
		resetEnv := prepTestEnv(t)

		env, err := newEnv()
		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, CsiMode, env.Mode)
		assert.True(t, env.CanFail)
		assert.NotEmpty(t, env.InstallerFlavor)
		assert.NotEmpty(t, env.InstallerTech)
		assert.NotEmpty(t, env.InstallerArch)
		assert.NotEmpty(t, env.InstallPath)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8Namespace)

		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)

		assert.True(t, env.OneAgentInjected)
		assert.True(t, env.DataIngestInjected)
	})
}

func prepTestEnv(t *testing.T) func() {
	envs := []string{
		InstallerFlavorEnv,
		InstallerTechEnv,
		InstallerArchEnv,
		K8NodeNameEnv,
		K8PodNameEnv,
		K8PodUIDEnv,
		K8BasePodNameEnv,
		K8NamespaceEnv,
		WorkloadKindEnv,
		WorkloadNameEnv,
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
	boolEnvs := []string{
		OneAgentInjectedEnv,
		DataIngestInjectedEnv,
	}
	for _, envvar := range boolEnvs {
		err := os.Setenv(envvar, "true")
		require.NoError(t, err)
	}
	envs = append(envs, boolEnvs...)

	return resetTestEnv(envs)
}

func resetTestEnv(envs []string) func() {
	return func() {
		for _, envvar := range envs {
			os.Unsetenv(envvar)
		}
	}
}
