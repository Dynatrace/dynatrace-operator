package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMutateUserContainers(t *testing.T) {
	dynakube := getTestDynakube()
	annotations := map[string]string{"container.inject.dyantrace/container": "false"}

	t.Run("Add volume mounts to containers", func(t *testing.T) {
		request := createTestMutationRequest(getTestDynakube(), nil, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in dynkube", func(t *testing.T) {
		dynakube.Annotations = annotations

		request := createTestMutationRequest(dynakube, nil, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in pod", func(t *testing.T) {
		request := createTestMutationRequest(dynakube, annotations, false)
		mutateUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})
}

func TestReinvokeUserContainers(t *testing.T) {
	dynakube := getTestDynakube()
	annotations := map[string]string{"container.inject.dyantrace/container": "false"}

	t.Run("Add volume mounts to containers", func(t *testing.T) {
		request := createTestReinvocationRequest(dynakube, nil)
		reinvokeUserContainers(request.BaseRequest)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in dynkube", func(t *testing.T) {
		dynakube.Annotations = annotations

		request := createTestReinvocationRequest(dynakube, nil)
		reinvokeUserContainers(request.BaseRequest)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{})
		reinvokeUserContainers(request.BaseRequest)

		for _, container := range request.Pod.Spec.Containers {
			require.GreaterOrEqual(t, len(container.VolumeMounts), 2)
		}
	})

	t.Run("Do not inject container if excluded in pod", func(t *testing.T) {
		request := createTestReinvocationRequest(dynakube, annotations)
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

		updateInstallContainer(container, createTestWorkloadInfo())

		require.Len(t, container.VolumeMounts, 1)
		require.Len(t, container.Env, 3)
	})
}
