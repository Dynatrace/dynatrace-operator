package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddWorkloadInfoEnvs(t *testing.T) {
	t.Run("Add workload info envs", func(t *testing.T) {
		container := &corev1.Container{}
		workloadInfo := createTestWorkloadInfo()
		addWorkloadInfoEnvs(container, workloadInfo)

		require.Len(t, container.Env, 2)
	})
}

func TestAddInjectedEnv(t *testing.T) {
	t.Run("Add workload info envs", func(t *testing.T) {
		container := &corev1.Container{}
		addInjectedEnv(container)

		require.Len(t, container.Env, 1)
	})
}

func TestAddClusterNameEnv(t *testing.T) {
	t.Run("Add workload info envs", func(t *testing.T) {
		container := &corev1.Container{}
		addClusterNameEnv(container, "test")

		require.Len(t, container.Env, 1)
	})
}
