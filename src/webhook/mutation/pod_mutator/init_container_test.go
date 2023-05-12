package pod_mutator

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
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
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, defaultUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, defaultGroup)

		require.NotNil(t, initContainer.SecurityContext.SeccompProfile)
		require.NotNil(t, initContainer.SecurityContext.SeccompProfile.Type)
		assert.Equal(t, initContainer.SecurityContext.SeccompProfile.Type, corev1.SeccompProfileTypeRuntimeDefault)
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
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, defaultUser)

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
	t.Run("should not set RunAsNonRoot if root user is used", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = address.Of(rootUserGroup)
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = address.Of(rootUserGroup)
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.Nil(t, initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, rootUserGroup)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, rootUserGroup)
	})
	t.Run("should handle failure policy feature flag correctly", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{v1beta1.AnnotationInjectionFailurePolicy: "fail"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.True(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "fail")
		assert.False(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "silent")
	})
	t.Run("should set default failure policy to silent", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{v1beta1.AnnotationInjectionFailurePolicy: "test"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.False(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "fail")
		assert.True(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "silent")
	})
	t.Run("should take silent as failure policy if set explicitly", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{v1beta1.AnnotationInjectionFailurePolicy: "silent"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.False(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "fail")
		assert.True(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "silent")
	})
	t.Run("should take pod annotation when set", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{v1beta1.AnnotationInjectionFailurePolicy: "silent"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "fail"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.True(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "fail")
		assert.False(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "silent")
	})
	t.Run("should fall back to feature flag if invalid value is set to pod annotation", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Annotations = map[string]string{v1beta1.AnnotationInjectionFailurePolicy: "fail"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "silent"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)

		assert.False(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "fail")
		assert.True(t, kubeobjects.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value == "silent")
	})
	t.Run("has seccompprofile set to RuntimeDefault", func(t *testing.T) {
		dynakube := getTestDynakube()
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dynakube)
		require.NotNil(t, initContainer.SecurityContext.SeccompProfile)
		require.NotNil(t, initContainer.SecurityContext.SeccompProfile.Type)
		assert.Equal(t, initContainer.SecurityContext.SeccompProfile.Type, corev1.SeccompProfileTypeRuntimeDefault)
	})
}
