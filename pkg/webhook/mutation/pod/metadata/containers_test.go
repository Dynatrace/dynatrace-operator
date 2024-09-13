package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMutateUserContainers(t *testing.T) {
	dk := getTestDynakube()
	annotations := map[string]string{"container.inject.dyantrace/container": "false"}

	t.Run("Add volume mounts to containers", func(t *testing.T) {
		request := createTestMutationRequest(getTestDynakube(), nil, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in dynkube", func(t *testing.T) {
		dk.Annotations = annotations

		request := createTestMutationRequest(dk, nil, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in pod", func(t *testing.T) {
		request := createTestMutationRequest(dk, annotations, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})
}

func TestReinvokeUserContainers(t *testing.T) {
	dk := getTestDynakube()
	annotations := map[string]string{"container.inject.dyantrace/container": "false"}

	t.Run("Add volume mounts to containers", func(t *testing.T) {
		request := createTestReinvocationRequest(dk, nil)
		reinvokeUserContainers(request.BaseRequest)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in dynkube", func(t *testing.T) {
		dk.Annotations = annotations

		request := createTestReinvocationRequest(dk, nil)
		reinvokeUserContainers(request.BaseRequest)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in pod", func(t *testing.T) {
		request := createTestReinvocationRequest(dk, annotations)
		reinvokeUserContainers(request.BaseRequest)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})
}

func TestUpdateInstallContainer(t *testing.T) {
	t.Run("Add volume mounts and envs", func(t *testing.T) {
		container := &corev1.Container{}
		clusterName := "test-cluster"
		entityID := "test-entity"

		updateInstallContainer(container, createTestWorkloadInfo(t), entityID, clusterName)

		require.Len(t, container.VolumeMounts, 1)
		require.Len(t, container.Env, 5)
	})
}
