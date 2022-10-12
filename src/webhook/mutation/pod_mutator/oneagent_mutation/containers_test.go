package oneagent_mutation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfigureInitContainer(t *testing.T) {
	t.Run("add envs and volume mounts (no-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, 6)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, config.AgentInstallModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, string(config.AgentInstallerMode), envvar.Value)
	})

	t.Run("add envs and volume mounts (csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestCSIDynakube(), nil, getTestNamespace(nil))
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Len(t, request.InstallContainer.Env, 6)
		assert.Len(t, request.InstallContainer.VolumeMounts, 2)
		envvar := kubeobjects.FindEnvVar(request.InstallContainer.Env, config.AgentInstallModeEnv)
		require.NotNil(t, envvar)
		assert.Equal(t, string(config.AgentCsiMode), envvar.Value)
	})
}

func TestMutateUserContainers(t *testing.T) {
	t.Run("add envs and volume mounts (simple dynakube)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil))
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
		request := createTestMutationRequest(getTestComplexDynakube(), nil, getTestNamespace(nil))
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
		request := createTestMutationRequest(getTestDynakube(), nil, getTestNamespace(nil)).ToReinvocationRequest()
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
		request := createTestMutationRequest(getTestComplexDynakube(), nil, getTestNamespace(nil)).ToReinvocationRequest()
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

func TestVersionDetectionMappingDrivenByNamespaceAnnotations(t *testing.T) {
	const (
		customVersionValue               = "my awesome custom version"
		customProductValue               = "my awesome custom product"
		customReleaseStageValue          = "my awesome custom stage"
		customBuildVersionValue          = "my awesome custom build version"
		customVersionAnnotationName      = "custom-version"
		customProductAnnotationName      = "custom-product"
		customStageAnnotationName        = "custom-stage"
		customBuildVersionAnnotationName = "custom-build-version"
		customVersionFieldPath           = "metadata.annotations['" + customVersionAnnotationName + "']"
		customProductFieldPath           = "metadata.annotations['" + customProductAnnotationName + "']"
		customStageFieldPath             = "metadata.annotations['" + customStageAnnotationName + "']"
		customBuildVersionFieldPath      = "metadata.annotations['" + customBuildVersionAnnotationName + "']"
	)

	t.Run("version and product env vars are set using values referenced in namespace annotations", func(t *testing.T) {
		podAnnotations := map[string]string{
			customVersionAnnotationName: customVersionValue,
			customProductAnnotationName: customProductValue,
		}
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
			productMappingAnnotationName: customProductFieldPath,
		}
		expectedMappings := map[string]string{
			releaseVersionEnv: customVersionFieldPath,
			releaseProductEnv: customProductFieldPath,
		}
		unexpectedMappingsKeys := []string{releaseStageEnv, releaseBuildVersionEnv}

		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
	})
	t.Run("only version env vars is set using value referenced in namespace annotations, product is default", func(t *testing.T) {
		podAnnotations := map[string]string{
			customVersionAnnotationName: customVersionValue,
		}
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
		}
		expectedMappings := map[string]string{
			releaseVersionEnv: customVersionFieldPath,
			releaseProductEnv: defaultVersionLabelMapping[releaseProductEnv],
		}
		unexpectedMappingsKeys := []string{releaseStageEnv, releaseBuildVersionEnv}

		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, unexpectedMappingsKeys)
	})
	t.Run("optional env vars (stage, build-version) are set using values referenced in namespace annotations, default ones remain default", func(t *testing.T) {
		podAnnotations := map[string]string{
			customStageAnnotationName:        customReleaseStageValue,
			customBuildVersionAnnotationName: customBuildVersionValue,
		}
		namespaceAnnotations := map[string]string{
			stageMappingAnnotationName: customStageFieldPath,
			buildVersionAnnotationName: customBuildVersionFieldPath,
		}
		expectedMappings := map[string]string{
			releaseVersionEnv:      defaultVersionLabelMapping[releaseVersionEnv],
			releaseProductEnv:      defaultVersionLabelMapping[releaseProductEnv],
			releaseStageEnv:        customStageFieldPath,
			releaseBuildVersionEnv: customBuildVersionFieldPath,
		}

		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, nil)
	})
	t.Run("all env vars are namespace-annotations driven", func(t *testing.T) {
		podAnnotations := map[string]string{
			customVersionAnnotationName:      customVersionValue,
			customProductAnnotationName:      customProductValue,
			customStageAnnotationName:        customReleaseStageValue,
			customBuildVersionAnnotationName: customBuildVersionValue,
		}
		namespaceAnnotations := map[string]string{
			versionMappingAnnotationName: customVersionFieldPath,
			productMappingAnnotationName: customProductFieldPath,
			stageMappingAnnotationName:   customStageFieldPath,
			buildVersionAnnotationName:   customBuildVersionFieldPath,
		}
		expectedMappings := map[string]string{
			releaseVersionEnv:      customVersionFieldPath,
			releaseProductEnv:      customProductFieldPath,
			releaseStageEnv:        customStageFieldPath,
			releaseBuildVersionEnv: customBuildVersionFieldPath,
		}

		doTestMappings(t, podAnnotations, namespaceAnnotations, expectedMappings, nil)
	})
}

func doTestMappings(t *testing.T, podAnnotations map[string]string, namespaceAnnotations map[string]string, expectedMappings map[string]string, unexpectedMappingsKeys []string) {
	mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
	request := createTestMutationRequest(getTestComplexDynakube(), podAnnotations, getTestNamespace(namespaceAnnotations))
	mutator.mutateUserContainers(request)

	assertContainsMappings(t, expectedMappings, request)
	assertDoesntContainMappings(t, unexpectedMappingsKeys, request)
}

func assertContainsMappings(t *testing.T, expectedMappings map[string]string, request *dtwebhook.MutationRequest) {
	for envName, fieldPath := range expectedMappings {
		assert.Contains(t, request.Pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name: envName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "",
					FieldPath:  fieldPath,
				},
			},
		})
	}
}

func assertDoesntContainMappings(t *testing.T, unexpectedMappingKeys []string, request *dtwebhook.MutationRequest) {
	for _, key := range unexpectedMappingKeys {
		idx := slices.IndexFunc(request.Pod.Spec.Containers[0].Env, func(envvar corev1.EnvVar) bool {
			return envvar.Name == key
		})
		assert.Negative(t, idx, key+" found in container's envvars")
	}
}
