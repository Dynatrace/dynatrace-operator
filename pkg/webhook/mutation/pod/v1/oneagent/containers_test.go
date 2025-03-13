package oneagent

import (
	"maps"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var expectedBaseInitContainerEnvCount = getInstallerInfoFieldCount() + 1 // +1 = oneagent-injected

func TestConfigureInitContainer(t *testing.T) {
	t.Run("add envs and volume mounts (no-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
	})

	t.Run("add envs and volume mounts (csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestCSIDynakube(), nil, getTestNamespace(nil))
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
	})
	t.Run("add envs and volume mounts (readonly-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestReadOnlyCSIDynakube(), nil, getTestNamespace(nil))
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, expectedBaseInitContainerEnvCount)
		assert.Len(t, request.InstallContainer.VolumeMounts, 3)
	})
}

type mutateUserContainerTestCase struct {
	name                               string
	dk                                 dynakube.DynaKube
	expectedAdditionalEnvCount         int
	expectedAdditionalVolumeMountCount int
}

func TestMutateUserContainers(t *testing.T) {
	testCases := []mutateUserContainerTestCase{
		{
			name:                               "add envs and volume mounts (simple dynakube)",
			dk:                                 *getTestDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 3, // 3 oneagent mounts(preload,bin,conf)
		},
		{
			name:                               "add envs and volume mounts (complex dynakube)",
			dk:                                 *getTestComplexDynakube(),
			expectedAdditionalEnvCount:         5, // 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
			expectedAdditionalVolumeMountCount: 5, // 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
		},
		{
			name:                               "add envs and volume mounts (readonly-csi)",
			dk:                                 *getTestReadOnlyCSIDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 6, // 3 oneagent mounts(preload,bin,conf) +3 oneagent mounts for readonly csi (agent-conf,data-storage,agent-log)
		},
	}
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
			request := createTestMutationRequest(&testCases[index].dk, nil, getTestNamespace(nil))
			initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
			initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)

			mutator.mutateUserContainers(request)

			assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+testCase.expectedAdditionalVolumeMountCount)
			assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+testCase.expectedAdditionalEnvCount)
		})
	}
}

func TestReinvokeUserContainers(t *testing.T) {
	testCases := []mutateUserContainerTestCase{
		{
			name:                               "add envs and volume mounts (simple dynakube)",
			dk:                                 *getTestDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 3, // 3 oneagent mounts(preload,bin,conf)
		},
		{
			name:                               "add envs and volume mounts (complex dynakube)",
			dk:                                 *getTestComplexDynakube(),
			expectedAdditionalEnvCount:         5, // 1 deployment-metadata + 1 network-zone + 1 preload + 2 version-detection
			expectedAdditionalVolumeMountCount: 5, // 3 oneagent mounts(preload,bin,conf) + 1 cert mount + 1 curl-options
		},
		{
			name:                               "add envs and volume mounts (readonly-csi)",
			dk:                                 *getTestReadOnlyCSIDynakube(),
			expectedAdditionalEnvCount:         2, // 1 deployment-metadata + 1 preload
			expectedAdditionalVolumeMountCount: 6, // 3 oneagent mounts(preload,bin,conf) +3 oneagent mounts for readonly csi (agent-conf,data-storage,agent-log)
		},
	}
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
			request := createTestMutationRequest(&testCases[index].dk, nil, getTestNamespace(nil)).ToReinvocationRequest()
			initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
			initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
			request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{
				Name: dtwebhook.InstallContainerName,
			})

			mutator.reinvokeUserContainers(request)
			request.Pod.Spec.Containers = append(request.Pod.Spec.Containers, corev1.Container{
				Name:  "test",
				Image: "test",
			})
			mutator.reinvokeUserContainers(request)

			assert.Len(t, request.Pod.Spec.Containers[0].VolumeMounts, initialContainerVolumeMountsLen+testCase.expectedAdditionalVolumeMountCount)
			assert.Len(t, request.Pod.Spec.Containers[0].Env, initialNumberOfContainerEnvsLen+testCase.expectedAdditionalEnvCount)
			assert.Len(t, request.Pod.Spec.Containers[2].VolumeMounts, testCase.expectedAdditionalVolumeMountCount)
			assert.Len(t, request.Pod.Spec.Containers[2].Env, testCase.expectedAdditionalEnvCount)
		})
	}
}

func TestContainerExclusion(t *testing.T) {
	testCases := []struct {
		name                               string
		dk                                 dynakube.DynaKube
		expectedAdditionalEnvCount         int
		expectedAdditionalVolumeMountCount int
		expectedInitContainerEnvCount      int
		dynakubeAnnotations                map[string]string
		podAnnotations                     map[string]string
	}{
		{
			name:                               "container exclusion on dynakube level",
			dk:                                 *getTestDynakubeWithContainerExclusion(),
			expectedAdditionalEnvCount:         0,
			expectedAdditionalVolumeMountCount: 0,
			expectedInitContainerEnvCount:      3,
			dynakubeAnnotations: map[string]string{
				dtwebhook.AnnotationContainerInjection + "/sidecar-container": "false",
			},
		},
		{
			name:                               "container exclusion on dynakube level, do not exclude",
			dk:                                 *getTestDynakubeWithContainerExclusion(),
			expectedAdditionalEnvCount:         2,
			expectedAdditionalVolumeMountCount: 3,
			expectedInitContainerEnvCount:      5,
			dynakubeAnnotations: map[string]string{
				dtwebhook.AnnotationContainerInjection + "/sidecar-container": "true",
			},
		},
		{
			name:                               "container exclusion on pod level",
			dk:                                 *getTestDynakube(),
			expectedAdditionalEnvCount:         0,
			expectedAdditionalVolumeMountCount: 0,
			expectedInitContainerEnvCount:      3,
			podAnnotations: map[string]string{
				dtwebhook.AnnotationContainerInjection + "/sidecar-container": "false",
			},
		},
		{
			name:                               "container exclusion on pod level, do not exclude",
			dk:                                 *getTestDynakube(),
			expectedAdditionalEnvCount:         2,
			expectedAdditionalVolumeMountCount: 3,
			expectedInitContainerEnvCount:      5,
			podAnnotations: map[string]string{
				dtwebhook.AnnotationContainerInjection + "/sidecar-container": "true",
			},
		},
	}

	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
			request := createTestMutationRequest(&testCases[index].dk, testCase.podAnnotations, getTestNamespace(nil)).ToReinvocationRequest()

			maps.Copy(request.DynaKube.Annotations, testCase.dynakubeAnnotations)

			initialNumberOfContainerEnvsLen := len(request.Pod.Spec.Containers[0].Env)
			initialContainerVolumeMountsLen := len(request.Pod.Spec.Containers[0].VolumeMounts)
			request.Pod.Spec.InitContainers = append(request.Pod.Spec.InitContainers, corev1.Container{
				Name: dtwebhook.InstallContainerName,
			})

			mutator.reinvokeUserContainers(request)

			assert.Len(t, request.Pod.Spec.Containers[1].VolumeMounts, initialContainerVolumeMountsLen+testCase.expectedAdditionalVolumeMountCount)
			assert.Len(t, request.Pod.Spec.Containers[1].Env, initialNumberOfContainerEnvsLen+testCase.expectedAdditionalEnvCount)
		})
	}
}
