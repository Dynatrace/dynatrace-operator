package v1

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestCreateInstallInitContainerBase(t *testing.T) {
	t.Run("should create the init container with set container sec ctx but without user and group", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = nil
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

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
		assert.Equal(t, oacommon.DefaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, oacommon.DefaultGroup, *initContainer.SecurityContext.RunAsGroup)

		assert.Nil(t, initContainer.SecurityContext.SeccompProfile)
	})
	t.Run("should overwrite partially", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		testUser := ptr.To(int64(420))
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, oacommon.DefaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, *testUser)
	})
	t.Run("container SecurityContext overrules defaults", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		overruledUser := ptr.To(int64(420))
		testUser := ptr.To(int64(420))
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = overruledUser
		pod.Spec.SecurityContext.RunAsGroup = overruledUser
		pod.Spec.Containers[0].SecurityContext.RunAsUser = testUser
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)
		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *initContainer.SecurityContext.RunAsUser, *testUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *initContainer.SecurityContext.RunAsGroup, *testUser)
	})
	t.Run("PodSecurityContext overrules defaults", func(t *testing.T) {
		dk := getTestDynakube()
		testUser := ptr.To(int64(420))
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext = nil
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = testUser
		pod.Spec.SecurityContext.RunAsGroup = testUser
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, *testUser, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should set RunAsNonRoot if root user is used", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = ptr.To(oacommon.RootUserGroup)
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = ptr.To(oacommon.RootUserGroup)
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.False(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, oacommon.RootUserGroup, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, oacommon.RootUserGroup, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should handle failure policy feature flag correctly", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "fail"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.Equal(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.NotEqual(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should set default failure policy to silent", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "test"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should take silent as failure policy if set explicitly", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "silent"}
		pod := getTestPod()
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should take pod annotation when set", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "silent"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "fail"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.Equal(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.NotEqual(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should fall back to feature flag if invalid value is set to pod annotation", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "fail"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		pod.Annotations[dtwebhook.AnnotationFailurePolicy] = "silent"
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.NotEqual(t, "fail", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
		assert.Equal(t, "silent", env.FindEnvVar(initContainer.Env, "FAILURE_POLICY").Value)
	})
	t.Run("should set seccomp profile if feature flag is enabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionSeccompKey: "true"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}
		webhookImage := "test-image"
		clusterID := "id"

		initContainer := createInstallInitContainerBase(webhookImage, clusterID, pod, *dk)

		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, initContainer.SecurityContext.SeccompProfile.Type)
	})
}

func TestInitContainerResources(t *testing.T) {
	t.Run("should return default if nothing is set", func(t *testing.T) {
		dk := getTestDynakubeNoInitLimits()

		initResources := initContainerResources(*dk)

		require.NotNil(t, initResources)
		assert.Equal(t, defaultInitContainerResources(), initResources)
	})

	t.Run("should return custom if set in dynakube", func(t *testing.T) {
		dk := getTestDynakube()

		initResources := initContainerResources(*dk)

		require.NotNil(t, initResources)
		assert.Equal(t, testResourceRequirements, initResources)
	})

	t.Run("should ignore if csi not used", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		dk := getTestDynakubeDefaultAppMon()

		initResources := initContainerResources(*dk)

		require.Empty(t, initResources)
	})

	t.Run("should have default if metadata enrichment is enabled", func(t *testing.T) {
		dk := getTestDynakubeDefaultAppMon()
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)

		initResources := initContainerResources(*dk)

		assert.Equal(t, defaultInitContainerResources(), initResources)
	})
	t.Run("should have default if only metadata enrichment is enabled and csi is disabled", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		dk := getTestDynakubeDefaultAppMon()
		dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
		dk.Spec.OneAgent.ApplicationMonitoring = nil
		initResources := initContainerResources(*dk)

		assert.Equal(t, defaultInitContainerResources(), initResources)
	})
	t.Run("should have no limit if OA is enabled and csi is disabled", func(t *testing.T) {
		installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

		dk := getTestDynakubeDefaultAppMon()

		dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
		dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}

		initResources := initContainerResources(*dk)

		require.Empty(t, initResources)
	})
}
