package oneagent

import (
	"testing"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestMutateUserContainers(t *testing.T) {
	t.Run("MutateUserContainersWithInjectedContainers", func(t *testing.T) {
		mutator := &Mutator{}
		request := createTestMutationRequestWithInjectedContainers()

		require.Len(t, request.NewContainers(ContainerIsInjected), 1)
		mutator.mutateUserContainers(request)
		require.Empty(t, request.NewContainers(ContainerIsInjected))
	})

	t.Run("MutateUserContainersWithoutInjectedContainers", func(t *testing.T) {
		mutator := &Mutator{}
		request := createTestMutationRequestWithoutInjectedContainers()

		require.Len(t, request.NewContainers(ContainerIsInjected), 2)
		mutator.mutateUserContainers(request)
		require.Empty(t, request.NewContainers(ContainerIsInjected))
	})
}

func TestReinvokeUserContainers(t *testing.T) {
	t.Run("ReinvokeUserContainersWithInjectedContainers", func(t *testing.T) {
		mutator := &Mutator{}
		request := createTestMutationRequestWithInjectedContainers().ToReinvocationRequest()

		require.Len(t, request.NewContainers(ContainerIsInjected), 1)
		require.True(t, mutator.reinvokeUserContainers(request))
		require.Empty(t, request.NewContainers(ContainerIsInjected))
		require.False(t, mutator.reinvokeUserContainers(request))
	})

	t.Run("ReinvokeUserContainersWithoutInjectedContainers", func(t *testing.T) {
		mutator := &Mutator{}
		request := createTestMutationRequestWithoutInjectedContainers().ToReinvocationRequest()

		require.Len(t, request.NewContainers(ContainerIsInjected), 2)
		require.True(t, mutator.reinvokeUserContainers(request))
		require.Empty(t, request.NewContainers(ContainerIsInjected))
	})
}

func createTestMutationRequestWithoutInjectedContainers() *dtwebhook.MutationRequest {
	return &dtwebhook.MutationRequest{
		BaseRequest: &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: dtwebhook.InstallContainerName,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "sample-container-1",
							Image: "sample-image-1",
						},
						{
							Name:  "sample-container-2",
							Image: "sample-image-2",
						},
					},
				},
			},
		},
	}
}

func createTestMutationRequestWithInjectedContainers() *dtwebhook.MutationRequest {
	request := createTestMutationRequestWithoutInjectedContainers()

	request.Pod.Spec.Volumes = append(request.Pod.Spec.Volumes, corev1.Volume{
		Name: "dynatrace-codemodules",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}, corev1.Volume{
		Name: "dynatrace-config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	i := 0
	request.Pod.Spec.Containers[i].Env = append(request.Pod.Spec.Containers[i].Env, corev1.EnvVar{
		Name:  "LD_PRELOAD",
		Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so",
	})
	request.Pod.Spec.Containers[i].VolumeMounts = append(request.Pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
		Name:      "dynatrace-codemodules",
		MountPath: "/opt/dynatrace/oneagent-paas",
	}, corev1.VolumeMount{
		Name:      "dynatrace-config",
		MountPath: "/var/lib/dynatrace",
	})

	return request
}
