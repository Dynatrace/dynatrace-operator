package startup

import (
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnv(t *testing.T) {
	t.Run(`create new env for oneagent and metadata-enrichment injection`, func(t *testing.T) {
		resetEnv := prepCombinedTestEnv(t)

		env, err := newEnv()

		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, failPhrase, env.FailurePolicy)
		assert.NotEmpty(t, env.InstallerFlavor)
		assert.NotEmpty(t, env.InstallerTech)
		assert.NotEmpty(t, env.InstallPath)
		assert.NotEmpty(t, env.InstallVersion)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8Namespace)
		assert.NotEmpty(t, env.K8ClusterID)

		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)
		assert.NotEmpty(t, env.K8ClusterName)

		assert.True(t, env.OneAgentInjected)
		assert.True(t, env.MetadataEnrichmentInjected)
	})
	t.Run(`create new env for only metadata-enrichment injection`, func(t *testing.T) {
		resetEnv := prepMetadataEnrichmentTestEnv(t, false)

		env, err := newEnv()

		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, failPhrase, env.FailurePolicy)
		assert.NotEmpty(t, env.InstallerFlavor) // set to what is defined in arch.Flavor
		assert.Empty(t, env.InstallerTech)
		assert.Empty(t, env.InstallVersion)
		assert.Empty(t, env.InstallPath)
		assert.Empty(t, env.Containers)

		assert.Empty(t, env.K8NodeName)
		assert.Empty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8Namespace)

		assert.NotEmpty(t, env.K8ClusterID)
		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)
		assert.NotEmpty(t, env.K8ClusterName)

		assert.False(t, env.OneAgentInjected)
		assert.True(t, env.MetadataEnrichmentInjected)
	})
	t.Run(`create new env for only oneagent`, func(t *testing.T) {
		resetEnv := prepOneAgentTestEnv(t)

		env, err := newEnv()

		resetEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, failPhrase, env.FailurePolicy)
		assert.NotEmpty(t, env.InstallerFlavor)
		assert.NotEmpty(t, env.InstallerTech)
		assert.NotEmpty(t, env.InstallVersion)
		assert.NotEmpty(t, env.InstallPath)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8Namespace)

		assert.NotEmpty(t, env.K8ClusterID)
		assert.Empty(t, env.WorkloadKind)
		assert.Empty(t, env.WorkloadName)
		assert.Empty(t, env.K8ClusterName)

		assert.True(t, env.OneAgentInjected)
		assert.False(t, env.MetadataEnrichmentInjected)
	})
}

func TestFailurePolicyModes(t *testing.T) {
	modes := map[string]string{
		failPhrase:   failPhrase,
		silentPhrase: silentPhrase,
		"Fail":       silentPhrase,
		"other":      silentPhrase,
	}
	for configuredMode, expectedMode := range modes {
		t.Run(`injection failure policy: `+configuredMode, func(t *testing.T) {
			resetEnv := prepMetadataEnrichmentTestEnv(t, true)

			t.Setenv(consts.InjectionFailurePolicyEnv, configuredMode)

			env, err := newEnv()

			resetEnv()

			require.NoError(t, err)
			require.NotNil(t, env)

			assert.Equal(t, expectedMode, env.FailurePolicy)
		})
	}
}

func prepCombinedTestEnv(t *testing.T) func() {
	resetMetadataEnrichmentEnvs := prepMetadataEnrichmentTestEnv(t, false)
	resetOneAgentEnvs := prepOneAgentTestEnv(t)

	return func() {
		resetMetadataEnrichmentEnvs()
		resetOneAgentEnvs()
	}
}

func prepOneAgentTestEnv(t *testing.T) func() {
	envs := []string{
		consts.AgentInstallerFlavorEnv,
		consts.AgentInstallerTechEnv,
		consts.AgentInstallerVersionEnv,
		consts.K8sNodeNameEnv,
		consts.K8sPodNameEnv,
		consts.K8sPodUIDEnv,
		consts.K8sBasePodNameEnv,
		consts.K8sNamespaceEnv,
		consts.AgentInstallPathEnv,
		consts.K8sClusterIDEnv,
	}
	for i := 1; i <= 5; i++ {
		envs = append(envs, fmt.Sprintf(consts.AgentContainerNameEnvTemplate, i))
		envs = append(envs, fmt.Sprintf(consts.AgentContainerImageEnvTemplate, i))
	}

	for _, envvar := range envs {
		err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		require.NoError(t, err)
	}

	// Int env
	envs = append(envs, consts.AgentContainerCountEnv)
	t.Setenv(consts.AgentContainerCountEnv, "5")

	// Mode Env
	envs = append(envs, consts.InjectionFailurePolicyEnv)
	t.Setenv(consts.InjectionFailurePolicyEnv, "fail")

	// Bool envs
	t.Setenv(consts.AgentInjectedEnv, trueStatement)

	envs = append(envs, consts.AgentInjectedEnv)

	return resetTestEnv(envs)
}

func prepMetadataEnrichmentTestEnv(t *testing.T, isUnknownWorkload bool) func() {
	envs := []string{
		consts.EnrichmentWorkloadKindEnv,
		consts.EnrichmentWorkloadNameEnv,
		consts.K8sClusterIDEnv,
		consts.K8sPodNameEnv,
		consts.K8sPodUIDEnv,
		consts.K8sNamespaceEnv,
		consts.EnrichmentClusterNameEnv,
	}
	for _, envvar := range envs {
		if isUnknownWorkload &&
			(envvar == consts.EnrichmentWorkloadKindEnv || envvar == consts.EnrichmentWorkloadNameEnv) {
			err := os.Setenv(envvar, "UNKNOWN")
			require.NoError(t, err)
		} else {
			err := os.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
			require.NoError(t, err)
		}
	}

	// Mode Env
	envs = append(envs, consts.InjectionFailurePolicyEnv)
	t.Setenv(consts.InjectionFailurePolicyEnv, "fail")

	// Bool envs
	t.Setenv(consts.EnrichmentInjectedEnv, "true")

	envs = append(envs, consts.EnrichmentInjectedEnv)

	return resetTestEnv(envs)
}

func resetTestEnv(envs []string) func() {
	return func() {
		for _, envvar := range envs {
			_ = os.Unsetenv(envvar)
		}
	}
}
