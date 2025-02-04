package startup

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnv(t *testing.T) {
	t.Run("create new env for oneagent and metadata-enrichment injection", func(t *testing.T) {
		prepCombinedTestEnv(t)

		env, err := newEnv()

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
	t.Run("create new env for only metadata-enrichment injection", func(t *testing.T) {
		prepMetadataEnrichmentTestEnv(t, false)

		env, err := newEnv()

		require.NoError(t, err)
		require.NotNil(t, env)

		assert.Equal(t, failPhrase, env.FailurePolicy)
		assert.NotEmpty(t, env.InstallerFlavor) // set to what is defined in arch.Flavor
		assert.Empty(t, env.InstallerTech)
		assert.Empty(t, env.InstallVersion)
		assert.Empty(t, env.InstallPath)
		assert.Len(t, env.Containers, 5)

		assert.NotEmpty(t, env.K8NodeName)
		assert.Empty(t, env.K8BasePodName)
		assert.NotEmpty(t, env.K8PodName)
		assert.NotEmpty(t, env.K8PodUID)
		assert.NotEmpty(t, env.K8Namespace)

		assert.NotEmpty(t, env.K8ClusterID)
		assert.NotEmpty(t, env.WorkloadKind)
		assert.NotEmpty(t, env.WorkloadName)
		assert.NotEmpty(t, env.WorkloadAnnotations)
		assert.NotEmpty(t, env.K8ClusterName)

		assert.False(t, env.OneAgentInjected)
		assert.True(t, env.MetadataEnrichmentInjected)
	})
	t.Run("create new env for only oneagent", func(t *testing.T) {
		prepOneAgentTestEnv(t)

		env, err := newEnv()

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
		t.Run("injection failure policy: "+configuredMode, func(t *testing.T) {
			prepMetadataEnrichmentTestEnv(t, true)

			t.Setenv(consts.InjectionFailurePolicyEnv, configuredMode)

			env, err := newEnv()

			require.NoError(t, err)
			require.NotNil(t, env)

			assert.Equal(t, expectedMode, env.FailurePolicy)
		})
	}
}

func prepCombinedTestEnv(t *testing.T) {
	prepMetadataEnrichmentTestEnv(t, false)
	prepOneAgentTestEnv(t)
}

func prepOneAgentTestEnv(t *testing.T) {
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

	for _, envvar := range envs {
		t.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
	}

	// Mode Env
	t.Setenv(consts.InjectionFailurePolicyEnv, "fail")

	// Bool envs
	t.Setenv(consts.AgentInjectedEnv, trueStatement)

	// Complex envs
	containerInfo := []ContainerInfo{}
	for i := 1; i <= 5; i++ {
		containerInfo = append(containerInfo, ContainerInfo{Name: fmt.Sprintf("TEST_CONTAINER_%d_NAME", i), Image: fmt.Sprintf("TEST_CONTAINER_%d_IMAGE", i)})
	}

	rawContainerInfo, err := json.Marshal(containerInfo)
	require.NoError(t, err)

	t.Setenv(consts.ContainerInfoEnv, string(rawContainerInfo))
}

func prepMetadataEnrichmentTestEnv(t *testing.T, isUnknownWorkload bool) {
	envs := []string{
		consts.EnrichmentWorkloadKindEnv,
		consts.EnrichmentWorkloadNameEnv,
		consts.K8sClusterIDEnv,
		consts.K8sPodNameEnv,
		consts.K8sPodUIDEnv,
		consts.K8sNodeNameEnv,
		consts.K8sNamespaceEnv,
		consts.EnrichmentClusterNameEnv,
		consts.EnrichmentClusterEntityIDEnv,
	}

	for _, envvar := range envs {
		if isUnknownWorkload &&
			(envvar == consts.EnrichmentWorkloadKindEnv || envvar == consts.EnrichmentWorkloadNameEnv) {
			t.Setenv(envvar, "UNKNOWN")
		} else {
			t.Setenv(envvar, fmt.Sprintf("TEST_%s", envvar))
		}
	}

	// Mode Env
	t.Setenv(consts.InjectionFailurePolicyEnv, "fail")

	// Bool envs
	t.Setenv(consts.EnrichmentInjectedEnv, "true")

	// Complex envs
	containerInfo := []ContainerInfo{}
	for i := 1; i <= 5; i++ {
		containerInfo = append(containerInfo, ContainerInfo{Name: fmt.Sprintf("app-%d", i), Image: fmt.Sprintf("image-%d", i)})
	}

	rawContainerInfo, err := json.Marshal(containerInfo)
	require.NoError(t, err)

	t.Setenv(consts.ContainerInfoEnv, string(rawContainerInfo))
	require.NoError(t, err)

	workloadAnnotations := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}

	rawWorkloadAnnotations, err := json.Marshal(workloadAnnotations)
	require.NoError(t, err)

	t.Setenv(consts.EnrichmentWorkloadAnnotationsEnv, string(rawWorkloadAnnotations))
	require.NoError(t, err)
}
