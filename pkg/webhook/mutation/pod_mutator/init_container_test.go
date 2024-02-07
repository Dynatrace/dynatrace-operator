package pod_mutator

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateInstallInitContainerBase(t *testing.T) {
	t.Run("should create the init container with set container sec ctx but without user and group", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = nil
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		require.NotNil(t, initContainer)
		assert.Equal(t, initContainer.Image, webhookImage)
		assert.Equal(t, initContainer.Resources, testResourceRequirements)

		require.NotNil(t, initContainer.SecurityContext.AllowPrivilegeEscalation)
		assert.False(t, *initContainer.SecurityContext.AllowPrivilegeEscalation)

		require.NotNil(t, initContainer.SecurityContext.Privileged)
		assert.False(t, *initContainer.SecurityContext.Privileged)

		require.NotNil(t, initContainer.SecurityContext.ReadOnlyRootFilesystem)
		assert.True(t, *initContainer.SecurityContext.ReadOnlyRootFilesystem)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, defaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, defaultGroup, *initContainer.SecurityContext.RunAsGroup)

		assert.Nil(t, initContainer.SecurityContext.SeccompProfile)
	})
	t.Run("should overwrite partially", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		testUser := address.Of(int64(420))
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, defaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, *testUser)
	})
	t.Run("container SecurityContext overrules defaults", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		overruledUser := address.Of(int64(420))
		testUser := address.Of(int64(420))
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = overruledUser
		pod.Spec.SecurityContext.RunAsGroup = overruledUser
		pod.Spec.Containers[0].SecurityContext.RunAsUser = testUser
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)
		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, *testUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, *testUser)
	})
	t.Run("PodSecurityContext overrules defaults", func(t *testing.T) {
		dynakube := getTestDynakube()
		testUser := address.Of(int64(420))
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext = nil
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = testUser
		pod.Spec.SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should set RunAsNonRoot if root user is used", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = address.Of(rootUserGroup)
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = address.Of(rootUserGroup)
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.False(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, rootUserGroup, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, rootUserGroup, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should handle failure policy feature flag correctly", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationInjectionFailurePolicy: "fail"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.Equal(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.NotEqual(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should set default failure policy to silent", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationInjectionFailurePolicy: "test"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should take silent as failure policy if set explicitly", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationInjectionFailurePolicy: "silent"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should take pod annotation when set", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationInjectionFailurePolicy: "silent"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "fail"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.Equal(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.NotEqual(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should fall back to feature flag if invalid value is set to pod annotation", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationInjectionFailurePolicy: "fail"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "silent"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should set seccomp profile if feature flag is enabled", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{dynatracev1beta1.AnnotationFeatureInitContainerSeccomp: "true"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, initContainer.SecurityContext.SeccompProfile.Type)
	})
}

func TestInitContainerResources(t *testing.T) {
	t.Run("should return default if nothing is set", func(t *testing.T) {
		dynakube := getTestDynakubeNoInitLimits()

		initResources := initContainerResources(*dynakube)

		require.NotNil(t, initResources)
		assert.Equal(t, defaultInitContainerResources(), initResources)
	})

	t.Run("should return custom if set in dynakube", func(t *testing.T) {
		dynakube := getTestDynakube()

		initResources := initContainerResources(*dynakube)

		require.NotNil(t, initResources)
		assert.Equal(t, testResourceRequirements, initResources)
	})

	t.Run("should ignore if csi not used", func(t *testing.T) {
		dynakube := getTestDynakubeDefaultAppMon()

		initResources := initContainerResources(*dynakube)

		require.Empty(t, initResources)
	})
}
