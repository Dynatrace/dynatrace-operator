package pod

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	volumeutils "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	webhookmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestCreateInitContainerBase(t *testing.T) {
	wh := createTestWebhook(t, webhookmock.NewMutator(t), webhookmock.NewMutator(t))

	t.Run("should create the init container with set container sec ctx but without user and group", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext.RunAsUser = nil
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = nil

		initContainer := wh.createInitContainerBase(pod, *dk)

		require.NotNil(t, initContainer)

		require.NotEmpty(t, initContainer.Args)
		assert.Equal(t, bootstrapper.Use, initContainer.Args[0])
		assert.Equal(t, dtwebhook.InstallContainerName, initContainer.Name)
		assert.Equal(t, initContainer.Image, wh.webhookPodImage)
		assert.Empty(t, initContainer.Resources)

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
	t.Run("do not take security context from user container", func(t *testing.T) {
		dk := getTestDynakube()
		pod := getTestPod()
		testUser := ptr.To(int64(420))
		pod.Spec.Containers[0].SecurityContext.RunAsUser = testUser
		pod.Spec.Containers[0].SecurityContext.RunAsGroup = testUser

		initContainer := wh.createInitContainerBase(pod, *dk)

		require.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.True(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, oacommon.DefaultUser, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, oacommon.DefaultGroup, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("PodSecurityContext overrules defaults", func(t *testing.T) {
		dk := getTestDynakube()
		testUser := ptr.To(int64(420))
		pod := getTestPod()
		pod.Spec.Containers[0].SecurityContext = nil
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = testUser
		pod.Spec.SecurityContext.RunAsGroup = testUser

		initContainer := wh.createInitContainerBase(pod, *dk)

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
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{}
		pod.Spec.SecurityContext.RunAsUser = ptr.To(oacommon.RootUserGroup)
		pod.Spec.SecurityContext.RunAsGroup = ptr.To(oacommon.RootUserGroup)

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotNil(t, initContainer.SecurityContext.RunAsNonRoot)
		assert.False(t, *initContainer.SecurityContext.RunAsNonRoot)

		require.NotNil(t, *initContainer.SecurityContext.RunAsUser)
		assert.Equal(t, oacommon.RootUserGroup, *initContainer.SecurityContext.RunAsUser)

		require.NotNil(t, *initContainer.SecurityContext.RunAsGroup)
		assert.Equal(t, oacommon.RootUserGroup, *initContainer.SecurityContext.RunAsGroup)
	})
	t.Run("should set seccomp profile if feature flag is enabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionSeccompKey: "true"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, initContainer.SecurityContext.SeccompProfile.Type)
	})

	t.Run("should not set suppress-error arg - according to dk", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "fail"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotContains(t, initContainer.Args, "--"+cmd.SuppressErrorsFlag)
	})

	t.Run("should not set suppress-error arg - according to pod", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "silent"}
		pod := getTestPod()
		pod.Annotations = map[string]string{dtwebhook.AnnotationFailurePolicy: "fail"}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.NotContains(t, initContainer.Args, "--"+cmd.SuppressErrorsFlag)
	})

	t.Run("should set suppress-error arg - default", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+cmd.SuppressErrorsFlag)
	})

	t.Run("should set suppress-error arg - unknown value", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Annotations = map[string]string{exp.InjectionFailurePolicyKey: "asd"}
		pod := getTestPod()
		pod.Annotations = map[string]string{}

		initContainer := wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+cmd.SuppressErrorsFlag)

		dk = getTestDynakube()
		dk.Annotations = map[string]string{}
		pod = getTestPod()
		pod.Annotations = map[string]string{dtwebhook.AnnotationFailurePolicy: "asd"}

		initContainer = wh.createInitContainerBase(pod, *dk)

		assert.Contains(t, initContainer.Args, "--"+cmd.SuppressErrorsFlag)
	})
}

func TestAddInitContainerToPod(t *testing.T) {
	t.Run("adds common volumes/mounts", func(t *testing.T) {
		pod := corev1.Pod{}
		initContainer := corev1.Container{}

		addInitContainerToPod(&pod, &initContainer)

		assert.Contains(t, pod.Spec.InitContainers, initContainer)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.True(t, volumeutils.IsIn(pod.Spec.Volumes, volumes.ConfigVolumeName))
		assert.True(t, volumeutils.IsIn(pod.Spec.Volumes, volumes.InputVolumeName))
		require.Len(t, initContainer.VolumeMounts, 2)
		assert.True(t, mounts.IsPathIn(initContainer.VolumeMounts, volumes.InitConfigMountPath))
		assert.True(t, mounts.IsPathIn(initContainer.VolumeMounts, volumes.InitInputMountPath))
	})
}

func TestInitContainerResources(t *testing.T) {
	testResourceRequirements := corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		}}

	t.Run("should return nothing per default", func(t *testing.T) {
		dk := getTestDynakube()

		initResources := initContainerResources(*dk)

		require.Empty(t, initResources)
	})

	t.Run("should return custom if set in dynakube", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.OneAgent = getAppMonSpec(&testResourceRequirements)

		initResources := initContainerResources(*dk)

		require.NotNil(t, initResources)
		assert.Equal(t, testResourceRequirements, initResources)
	})
}

func getAppMonSpec(initResources *corev1.ResourceRequirements) oneagent.Spec {
	return oneagent.Spec{
		ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
			AppInjectionSpec: oneagent.AppInjectionSpec{
				InitResources:    initResources,
				CodeModulesImage: customImage,
			}},
	}
}
