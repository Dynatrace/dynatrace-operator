package oneagent

import (
	"maps"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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

//  TODO fix or move
// func TestVersionDetectionMappingDrivenByNamespaceAnnotations(t *testing.T) {
// 	const (
// 		customVersionValue               = "my awesome custom version"
// 		customProductValue               = "my awesome custom product"
// 		customReleaseStageValue          = "my awesome custom stage"
// 		customBuildVersionValue          = "my awesome custom build version"
// 		customVersionAnnotationName      = "custom-version"
// 		customProductAnnotationName      = "custom-product"
// 		customStageAnnotationName        = "custom-stage"
// 		customBuildVersionAnnotationName = "custom-build-version"
// 		customVersionFieldPath           = "metadata.podAnnotations['" + customVersionAnnotationName + "']"
// 		customProductFieldPath           = "metadata.podAnnotations['" + customProductAnnotationName + "']"
// 		customStageFieldPath             = "metadata.podAnnotations['" + customStageAnnotationName + "']"
// 		customBuildVersionFieldPath      = "metadata.podAnnotations['" + customBuildVersionAnnotationName + "']"
// 	)
//
// 	t.Run("version and product env vars are set using values referenced in namespace podAnnotations", func(t *testing.T) {
// 		podAnnotations := map[string]string{
// 			customVersionAnnotationName: customVersionValue,
// 			customProductAnnotationName: customProductValue,
// 		}
// 		namespaceAnnotations := map[string]string{
// 			versionMappingAnnotationName: customVersionFieldPath,
// 			productMappingAnnotationName: customProductFieldPath,
// 		}
// 		expectedMappings := map[string]string{
// 			releaseVersionEnv: customVersionFieldPath,
// 			releaseProductEnv: customProductFieldPath,
// 		}
// 		unexpectedMappingsKeys := []string{releaseStageEnv, releaseBuildVersionEnv}
//
// 		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
// 	})
// 	t.Run("only version env vars is set using value referenced in namespace podAnnotations, product is default", func(t *testing.T) {
// 		podAnnotations := map[string]string{
// 			customVersionAnnotationName: customVersionValue,
// 		}
// 		namespaceAnnotations := map[string]string{
// 			versionMappingAnnotationName: customVersionFieldPath,
// 		}
// 		expectedMappings := map[string]string{
// 			releaseVersionEnv: customVersionFieldPath,
// 			releaseProductEnv: defaultVersionLabelMapping[releaseProductEnv],
// 		}
// 		unexpectedMappingsKeys := []string{releaseStageEnv, releaseBuildVersionEnv}
//
// 		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
// 	})
// 	t.Run("optional env vars (stage, build-version) are set using values referenced in namespace podAnnotations, default ones remain default", func(t *testing.T) {
// 		podAnnotations := map[string]string{
// 			customStageAnnotationName:        customReleaseStageValue,
// 			customBuildVersionAnnotationName: customBuildVersionValue,
//		}
//		namespaceAnnotations := map[string]string{
//			stageMappingAnnotationName: customStageFieldPath,
//			buildVersionAnnotationName: customBuildVersionFieldPath,
//		}
//		expectedMappings := map[string]string{
//			releaseVersionEnv:      defaultVersionLabelMapping[releaseVersionEnv],
//			releaseProductEnv:      defaultVersionLabelMapping[releaseProductEnv],
//			releaseStageEnv:        customStageFieldPath,
//			releaseBuildVersionEnv: customBuildVersionFieldPath,
//		}
//
//		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, nil)
//	})
//	t.Run("all env vars are namespace-podAnnotations driven", func(t *testing.T) {
//		podAnnotations := map[string]string{
//			customVersionAnnotationName:      customVersionValue,
//			customProductAnnotationName:      customProductValue,
//			customStageAnnotationName:        customReleaseStageValue,
//			customBuildVersionAnnotationName: customBuildVersionValue,
//		}
//		namespaceAnnotations := map[string]string{
//			versionMappingAnnotationName: customVersionFieldPath,
//			productMappingAnnotationName: customProductFieldPath,
//			stageMappingAnnotationName:   customStageFieldPath,
//			buildVersionAnnotationName:   customBuildVersionFieldPath,
//		}
//		expectedMappings := map[string]string{
//			releaseVersionEnv:      customVersionFieldPath,
//			releaseProductEnv:      customProductFieldPath,
//			releaseStageEnv:        customStageFieldPath,
//			releaseBuildVersionEnv: customBuildVersionFieldPath,
//		}
//
//		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, nil)
//	})
// }

// func doTestMappings(t *testing.T, podAnnotations map[string]string, namespaceAnnotations map[string]string, expectedMappings map[string]string, unexpectedMappingsKeys []string) {
//	mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
//	request := createTestMutationRequest(getTestComplexDynakube(), podAnnotations, getTestNamespace(namespaceAnnotations))
//	mutator.mutateUserContainers(request)
//
//	assertContainsMappings(t, expectedMappings, request)
//	assertNotContainsMappings(t, unexpectedMappingsKeys, request)
// }
//
// func assertContainsMappings(t *testing.T, expectedMappings map[string]string, request *dtwebhook.MutationRequest) {
//	for envName, fieldPath := range expectedMappings {
//		assert.Contains(t, request.Pod.Spec.Containers[0].Env, corev1.EnvVar{
//			Name: envName,
//			ValueFrom: &corev1.EnvVarSource{
//				FieldRef: &corev1.ObjectFieldSelector{
//					APIVersion: "",
//					FieldPath:  fieldPath,
//				},
//			},
//		})
//	}
// }
//
// func assertNotContainsMappings(t *testing.T, unexpectedMappingKeys []string, request *dtwebhook.MutationRequest) {
//	for _, env := range request.Pod.Spec.Containers[0].Env {
//		assert.NotContains(t, unexpectedMappingKeys, env.Name)
//	}
// }
