package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetContainer(t *testing.T) {
	t.Run("get main container", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		mainContainer := getContainer(dk)

		require.NotEmpty(t, mainContainer)

		assert.NotEmpty(t, mainContainer.Name)
		assert.NotEmpty(t, mainContainer.Image)
		assert.NotEmpty(t, mainContainer.ImagePullPolicy)
		assert.NotEmpty(t, mainContainer.VolumeMounts)
		assert.Len(t, mainContainer.VolumeMounts, expectedMountLen)
		assert.NotEmpty(t, mainContainer.Env)
		assert.Len(t, mainContainer.Env, expectedBaseEnvLen)
		assert.Empty(t, mainContainer.Resources)
		assert.NotEmpty(t, mainContainer.SecurityContext)
	})

	t.Run("image-ref is respected", func(t *testing.T) {
		expectedRepo := "my-test-repo"
		expectedTag := "my-test-tag"
		dk := dynakube.DynaKube{}
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			ImageRef: image.Ref{
				Repository: expectedRepo,
				Tag:        expectedTag,
			},
		}
		mainContainer := getContainer(dk)

		require.NotEmpty(t, mainContainer)
		assert.NotEmpty(t, mainContainer.Image)
		assert.Equal(t, expectedRepo+":"+expectedTag, mainContainer.Image)
	})

	t.Run("resources are respected", func(t *testing.T) {
		requests := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
		limits := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		}
		dk := dynakube.DynaKube{}
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Resources: corev1.ResourceRequirements{
				Requests: requests,
				Limits:   limits,
			},
		}
		mainContainer := getContainer(dk)

		require.NotEmpty(t, mainContainer)
		assert.NotEmpty(t, mainContainer.Resources)
		assert.Equal(t, requests, mainContainer.Resources.Requests)
		assert.Equal(t, limits, mainContainer.Resources.Limits)
	})
}

func TestGetInitContainer(t *testing.T) {
	t.Run("get main container", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"

		initContainer := getInitContainer(dk)

		require.NotEmpty(t, initContainer)

		assert.NotEmpty(t, initContainer.Name)
		assert.NotEmpty(t, initContainer.Image)
		assert.NotEmpty(t, initContainer.ImagePullPolicy)
		assert.NotEmpty(t, initContainer.Args)
		assert.Len(t, initContainer.Args, expectedBaseInitArgsLen)
		assert.NotEmpty(t, initContainer.Command)
		assert.NotEmpty(t, initContainer.VolumeMounts)
		assert.Len(t, initContainer.VolumeMounts, expectedInitMountLen)
		assert.NotEmpty(t, initContainer.Env)
		assert.Len(t, initContainer.Env, expectedBaseInitEnvLen)
		assert.NotEmpty(t, initContainer.SecurityContext)
	})

	t.Run("image-ref is respected", func(t *testing.T) {
		expectedRepo := "my-test-repo"
		expectedTag := "my-test-tag"
		dk := dynakube.DynaKube{}
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			ImageRef: image.Ref{
				Repository: expectedRepo,
				Tag:        expectedTag,
			},
		}
		initContainer := getInitContainer(dk)

		require.NotEmpty(t, initContainer)
		assert.NotEmpty(t, initContainer.Image)
		assert.Equal(t, expectedRepo+":"+expectedTag, initContainer.Image)
	})

	t.Run("resources are respected", func(t *testing.T) {
		requests := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		}
		limits := corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		}
		dk := dynakube.DynaKube{}
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Resources: corev1.ResourceRequirements{
				Requests: requests,
				Limits:   limits,
			},
		}
		initContainer := getInitContainer(dk)

		require.NotEmpty(t, initContainer)
		assert.NotEmpty(t, initContainer.Resources)
		assert.Equal(t, requests, initContainer.Resources.Requests)
		assert.Equal(t, limits, initContainer.Resources.Limits)
	})

	t.Run("get main container without the use of metadata", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		initContainer := getInitContainer(dk)

		require.NotEmpty(t, initContainer)

		assert.NotEmpty(t, initContainer.Args)
		assert.Len(t, initContainer.Args, expectedBaseInitArgsLenWithoutMEID)
		assert.NotEmpty(t, initContainer.Env)
		assert.Len(t, initContainer.Env, expectedBaseInitEnvLenWithoutMEID)
	})
}

func TestSecurityContext(t *testing.T) {
	t.Run("get base securityContext", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		sc := getBaseSecurityContext(dk)

		require.NotNil(t, sc)
		require.NotEmpty(t, sc)

		assert.NotNil(t, sc.Privileged)
		assert.NotNil(t, sc.ReadOnlyRootFilesystem)
		assert.NotNil(t, sc.AllowPrivilegeEscalation)
		assert.NotNil(t, sc.RunAsUser)
		assert.NotNil(t, sc.RunAsGroup)
		assert.NotNil(t, sc.RunAsNonRoot)
		assert.NotNil(t, sc.Capabilities)

		assert.Nil(t, sc.SeccompProfile)
	})

	t.Run("set seccomp is present", func(t *testing.T) {
		expectedSeccomp := "test-seccomp"
		dk := dynakube.DynaKube{}
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{SecCompProfile: expectedSeccomp}
		sc := getBaseSecurityContext(dk)

		require.NotNil(t, sc)
		require.NotEmpty(t, sc)

		require.NotNil(t, sc.SeccompProfile)
		require.NotNil(t, sc.SeccompProfile.LocalhostProfile)
		assert.Equal(t, expectedSeccomp, *sc.SeccompProfile.LocalhostProfile)
	})

	t.Run("main and init container securityContext differ only in capabilities", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		initContainer := getInitContainer(dk)
		mainContainer := getContainer(dk)

		require.NotNil(t, initContainer)
		require.NotNil(t, mainContainer)

		assert.NotEqual(t, *initContainer.SecurityContext, *mainContainer.SecurityContext)
		initContainer.SecurityContext.Capabilities = nil
		mainContainer.SecurityContext.Capabilities = nil
		assert.Equal(t, *initContainer.SecurityContext, *mainContainer.SecurityContext)
	})

	t.Run("ocp scenario, OA needs to be privileged", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Annotations = map[string]string{
			exp.OAPrivilegedKey: "true",
		}

		sc := getBaseSecurityContext(dk)

		require.NotNil(t, sc)
		require.NotEmpty(t, sc)
		assert.True(t, *sc.Privileged)
		assert.True(t, *sc.AllowPrivilegeEscalation)
	})
}
