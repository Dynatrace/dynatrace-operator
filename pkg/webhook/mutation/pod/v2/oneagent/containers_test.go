package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestContainerIsInjected(t *testing.T) {
	t.Run("is injected", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			assert.False(t, containerIsInjected(*container))
			setIsInjectedEnv(container)
			assert.True(t, containerIsInjected(*container))
		}
	})
}

func TestMutate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()

		original := createTestMutationRequestWithoutInjectedContainers()
		updated := Mutate(request)
		require.True(t, updated)
		// update install container
		assert.NotEqual(t, original.InstallContainer, request.InstallContainer)

		for i := range request.Pod.Spec.Containers {
			// update each container
			assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i]))
		}
	})
	t.Run("install-path respected", func(t *testing.T) {
		expectedInstallPath := "my-install"
		request := createTestMutationRequestWithoutInjectedContainers()
		request.Pod.Annotations = map[string]string{
			oacommon.AnnotationInstallPath: expectedInstallPath,
		}

		updated := Mutate(request)
		require.True(t, updated)

		assert.Contains(t, request.InstallContainer.Args, "--"+configure.InstallPathFlag+"="+expectedInstallPath)

		for _, c := range request.Pod.Spec.Containers {
			preload := env.FindEnvVar(c.Env, oacommon.PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})
	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			setIsInjectedEnv(container)
			updateContainer = append(updateContainer, *container)
		}

		request.Pod.Spec.Containers = updateContainer

		updated := Mutate(request)
		require.False(t, updated)
	})
}

func TestReinvoke(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithInjectedContainers()

		original := createTestMutationRequestWithInjectedContainers()
		updated := Reinvoke(request.BaseRequest)
		require.True(t, updated)

		// no update to install container
		assert.Equal(t, original.InstallContainer, request.InstallContainer)

		for i := range request.Pod.Spec.Containers {
			// only update not-injected
			if containerIsInjected(original.Pod.Spec.Containers[i]) {
				assert.Equal(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			} else {
				assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			}

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i]))
		}
	})

	t.Run("install-path respected", func(t *testing.T) {
		expectedInstallPath := "my-install"
		request := createTestMutationRequestWithoutInjectedContainers()
		request.Pod.Annotations = map[string]string{
			oacommon.AnnotationInstallPath: expectedInstallPath,
		}

		updated := Reinvoke(request.BaseRequest)
		require.True(t, updated)

		for _, c := range request.Pod.Spec.Containers {
			preload := env.FindEnvVar(c.Env, oacommon.PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})

	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			setIsInjectedEnv(container)
			updateContainer = append(updateContainer, *container)
		}

		request.Pod.Spec.Containers = updateContainer

		updated := Reinvoke(request.BaseRequest)
		require.False(t, updated)
	})
}

func TestAddOneAgentToContainer(t *testing.T) {
	kubeSystemUUID := "my uuid"
	networkZone := "my zone"
	installPath := "install/path"

	t.Run("add everything", func(t *testing.T) {
		container := corev1.Container{}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent:    oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}},
				NetworkZone: networkZone,
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: kubeSystemUUID,
			},
		}

		addOneAgentToContainer(dk, &container, corev1.Namespace{}, installPath)

		assert.Len(t, container.VolumeMounts, 3) // preload,bin,config

		dtMetaEnv := env.FindEnvVar(container.Env, oacommon.DynatraceMetadataEnv)
		require.NotNil(t, dtMetaEnv)
		assert.Contains(t, dtMetaEnv.Value, kubeSystemUUID)

		dtZoneEnv := env.FindEnvVar(container.Env, oacommon.NetworkZoneEnv)
		require.NotNil(t, dtZoneEnv)
		assert.Equal(t, networkZone, dtZoneEnv.Value)

		preload := env.FindEnvVar(container.Env, oacommon.PreloadEnv)
		require.NotNil(t, preload)
		assert.Contains(t, preload.Value, installPath)

		assert.True(t, containerIsInjected(container))
	})
}

func createTestMutationRequestWithoutInjectedContainers() *dtwebhook.MutationRequest {
	return &dtwebhook.MutationRequest{
		InstallContainer: &corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		BaseRequest: &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
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

	i := 0
	request.Pod.Spec.Containers[i].Env = append(request.Pod.Spec.Containers[i].Env, corev1.EnvVar{
		Name:  isInjectedEnv,
		Value: "true",
	})

	return request
}
