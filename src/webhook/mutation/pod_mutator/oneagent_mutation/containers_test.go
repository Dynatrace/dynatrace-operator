package oneagent_mutation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfigureInitContainer(t *testing.T) {
	t.Run("add envs and volume mounts (no-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, 6)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, standalone.ModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, installerVolumeMode, envvar.Value)
	})

	t.Run("add envs and volume mounts (csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestCSIDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, 6)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, standalone.ModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, provisionedVolumeMode, envvar.Value)
	})
}

func TestMutateUserContainers(t *testing.T) {
	t.Run("add envs and volume mounts (simple dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		mutator.mutateUserContainers(request)

		require.Len(t, request.InstallContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, request.InstallContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, request.InstallContainer.Env[1].Value)
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+2)
	})

	t.Run("add envs and volume mounts (complex dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestComplexDynakube(), nil)
		initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
		initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

		mutator.mutateUserContainers(request)

		require.Len(t, request.InstallContainer.Env, len(request.Pod.Spec.Containers)*2)
		assert.Equal(t, request.Pod.Spec.Containers[0].Name, request.InstallContainer.Env[0].Value)
		assert.Equal(t, request.Pod.Spec.Containers[0].Image, request.InstallContainer.Env[1].Value)

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+4)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+5)
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
		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+3)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+2)
		assert.Len(t, request.Pod.Spec.Containers[1].VolumeMounts, 3)
		assert.Len(t, request.Pod.Spec.Containers[1].Env, 2)
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

		assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+4)
		assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+5)
		assert.Len(t, request.Pod.Spec.Containers[1].VolumeMounts, 4)
		assert.Len(t, request.Pod.Spec.Containers[1].Env, 5)
	})
}
