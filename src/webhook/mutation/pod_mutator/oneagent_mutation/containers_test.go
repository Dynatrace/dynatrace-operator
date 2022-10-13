package oneagent_mutation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var expectedBaseInitContainerEnvCount = getInstallerInfoFieldCount() + 2 // volumeMode + oneagent injected

func TestConfigureInitContainer(t *testing.T) {
	t.Run("add envs and volume mounts (no-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, config.AgentInstallModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, string(config.AgentInstallerMode), envvar.Value)
	})

	t.Run("add envs and volume mounts (csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestCSIDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, config.AgentInstallModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, string(config.AgentCsiMode), envvar.Value)
	})
}

func TestMutateUserContainers(t *testing.T) {
	t.Run("add envs and volume mounts (simple dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		// 1 deployment-metadata + 1 preload
		expectedAdditionalEnvCount := 2

		// 3 oneagent mounts(preload,bin,conf)
		expectedAdditionalVolumeCount := 3

		mutator.mutateUserContainers(request)

		require.Len(t, request.InstallContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, request.InstallContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, request.InstallContainer.Env[1].Value)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+expectedAdditionalEnvCount)
	})

	t.Run("add envs and volume mounts (complex dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestComplexDynakube(), nil)
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		// 1 proxy + 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
		expectedAdditionalEnvCount := 6

		// 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
		expectedAdditionalVolumeCount := 5

		mutator.mutateUserContainers(request)

		require.Len(t, request.InstallContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, request.InstallContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, request.InstallContainer.Env[1].Value)

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+expectedAdditionalEnvCount)
	})
}

func TestReinvokeUserContainers(t *testing.T) {
	t.Run("add envs and volume mounts (simple dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil).ToReinvocationRequest()
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{
			Name: dtwebhook.InstallContainerName,
		})
		installContainer := &request.Pod.Spec.InitContainers[1]

		// 1 deployment-metadata + 1 preload
		expectedAdditionalEnvCount := 2

		// 3 oneagent(preload,bin,conf) mounts
		expectedAdditionalVolumeCount := 3

		mutator.reinvokeUserContainers(request)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{
			Name:  "test",
			Image: "test",
		})
		mutator.reinvokeUserContainers(request)

		require.Len(t, installContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, installContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, installContainer.Env[1].Value)
		assert.Equal(t, request.Pod.Spec.Containers[1].Name, installContainer.Env[2].Value)
		assert.Equal(t, request.Pod.Spec.Containers[1].Image, installContainer.Env[3].Value)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+expectedAdditionalEnvCount)
		assert.Len(t, request.Pod.Spec.Containers[1].VolumeMounts, expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[1].Env, expectedAdditionalEnvCount)
	})

	t.Run("add envs and volume mounts (complex dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestComplexDynakube(), nil).ToReinvocationRequest()
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
		request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{
			Name: dtwebhook.InstallContainerName,
		})
		installContainer := &request.Pod.Spec.InitContainers[1]

		// 1 proxy + 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
		expectedAdditionalEnvCount := 6

		// 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
		expectedAdditionalVolumeCount := 5

		mutator.reinvokeUserContainers(request)
		request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{
			Name:  "test",
			Image: "test",
		})
		mutator.reinvokeUserContainers(request)

		require.Len(t, installContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, installContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, installContainer.Env[1].Value)
		assert.Equal(t, request.Pod.Spec.Containers[1].Name, installContainer.Env[2].Value)
		assert.Equal(t, request.Pod.Spec.Containers[1].Image, installContainer.Env[3].Value)

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+expectedAdditionalEnvCount)
		assert.Len(t, request.Pod.Spec.Containers[1].VolumeMounts, expectedAdditionalVolumeCount)
		assert.Len(t, request.Pod.Spec.Containers[1].Env, expectedAdditionalEnvCount)
	})
}
