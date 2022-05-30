package dataingest_mutation

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMutateUserContainers(t *testing.T) {
	t.Run("Add volume mounts to containers", func(t *testing.T) {
		pod := getTestPod(nil)

		mutateUserContainers(pod)

		for _, container := range pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})
}

func TestReinvokeUserContainers(t *testing.T) {
	t.Run("Add volume mounts to containers", func(t *testing.T) {
		pod := getTestPod(nil)

		reinvokeUserContainers(pod)
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(pod)

		for _, container := range pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})
}

func TestUpdateInstallContainer(t *testing.T) {
	t.Run("Add volume mounts and envs", func(t *testing.T) {
		container := &corev1.Container{}

		updateInstallContainer(container, createTestWorkloadInfo())

		require.Len(t, container.VolumeMounts, 1)
		require.Len(t, container.Env, 3)
	})
}
