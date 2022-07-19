package daemonset

import (
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testImageTag = "tag"
	testImage    = "test-image:" + testImageTag
	testVersion  = "test-version"
)

func TestUseImmutableImage(t *testing.T) {
	t.Run(`if image is unset, image`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, instance.ImmutableOneAgentImage(), podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set, set image is used`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						Image: testImage,
					},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
}

func TestLabels(t *testing.T) {
	feature := strings.ReplaceAll(DeploymentTypeFullStack, "_", "")
	t.Run(`if image is unset, use version`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						Version: testVersion,
					},
				},
			},
		}
		expectedLabels := map[string]string{
			kubeobjects.AppNameLabel:      kubeobjects.OneAgentComponentLabel,
			kubeobjects.AppCreatedByLabel: instance.Name,
			kubeobjects.AppComponentLabel: feature,
			kubeobjects.AppVersionLabel:   testVersion,
			kubeobjects.AppManagedByLabel: version.AppName,
		}
		expectedMatchLabels := map[string]string{
			kubeobjects.AppNameLabel:      kubeobjects.OneAgentComponentLabel,
			kubeobjects.AppCreatedByLabel: instance.Name,
			kubeobjects.AppManagedByLabel: version.AppName,
		}
		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, instance.ImmutableOneAgentImage(), podSpecs.Containers[0].Image)
		assert.Equal(t, expectedLabels, ds.Labels)
		assert.Equal(t, expectedMatchLabels, ds.Spec.Selector.MatchLabels)
		assert.Equal(t, expectedLabels, ds.Spec.Template.Labels)
	})
	t.Run(`if image is set, set basic version label`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						Image: testImage,
					},
				},
			},
		}

		expectedLabels := map[string]string{
			kubeobjects.AppNameLabel:      kubeobjects.OneAgentComponentLabel,
			kubeobjects.AppCreatedByLabel: instance.Name,
			kubeobjects.AppComponentLabel: feature,
			kubeobjects.AppManagedByLabel: version.AppName,
			kubeobjects.AppVersionLabel:   kubeobjects.CustomImageLabelValue,
		}
		expectedMatchLabels := map[string]string{
			kubeobjects.AppNameLabel:      kubeobjects.OneAgentComponentLabel,
			kubeobjects.AppCreatedByLabel: instance.Name,
			kubeobjects.AppManagedByLabel: version.AppName,
		}

		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
		assert.Equal(t, expectedLabels, ds.Labels)
		assert.Equal(t, expectedMatchLabels, ds.Spec.Selector.MatchLabels)
		assert.Equal(t, expectedLabels, ds.Spec.Template.Labels)

	})
}

func TestCustomPullSecret(t *testing.T) {
	instance := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testURL,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
			CustomPullSecret: testName,
		},
	}
	dsInfo := NewClassicFullStack(&instance, testClusterID)
	ds, err := dsInfo.BuildDaemonSet()
	require.NoError(t, err)

	podSpecs := ds.Spec.Template.Spec
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.ImagePullSecrets)
	assert.Equal(t, testName, podSpecs.ImagePullSecrets[0].Name)
}

func TestResources(t *testing.T) {
	t.Run(`minimal cpu request of 100mC is set if no resources specified`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasMinimumCPURequest := resource.NewScaledQuantity(1, -1).Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		assert.True(t, hasMinimumCPURequest)
	})
	t.Run(`resource requests and limits set`, func(t *testing.T) {
		cpuRequest := resource.NewScaledQuantity(2, -1)
		cpuLimit := resource.NewScaledQuantity(3, -1)
		memoryRequest := resource.NewScaledQuantity(1, 3)
		memoryLimit := resource.NewScaledQuantity(2, 3)

		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						OneAgentResources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *cpuRequest,
								corev1.ResourceMemory: *memoryRequest,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *cpuLimit,
								corev1.ResourceMemory: *memoryLimit,
							},
						},
					},
				},
			},
		}

		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasCPURequest := cpuRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		hasCPULimit := cpuLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Cpu())
		hasMemoryRequest := memoryRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Memory())
		hasMemoryLimit := memoryLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Memory())

		assert.True(t, hasCPURequest)
		assert.True(t, hasCPULimit)
		assert.True(t, hasMemoryRequest)
		assert.True(t, hasMemoryLimit)
	})
}

func TestHostMonitoring_SecurityContext(t *testing.T) {
	t.Run(`User and group id set when read only mode is enabled`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewHostMonitoring(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Equal(t, address.Of(int64(1000)), securityContext.RunAsUser)
		assert.Equal(t, address.Of(int64(1000)), securityContext.RunAsGroup)
		assert.Equal(t, address.Of(true), securityContext.RunAsNonRoot)
		assert.NotEmpty(t, securityContext.Capabilities)
	})

	t.Run(`No User and group id set when read only mode is disabled`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "false",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewHostMonitoring(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Nil(t, securityContext.RunAsUser)
		assert.Nil(t, securityContext.RunAsGroup)
		assert.Nil(t, securityContext.RunAsNonRoot)
		assert.Nil(t, securityContext.Privileged)
		assert.NotEmpty(t, securityContext.Capabilities)
	})

	t.Run(`privileged security context when feature flag is enabled`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureRunOneAgentContainerPrivileged: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewHostMonitoring(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Equal(t, address.Of(int64(1000)), securityContext.RunAsUser)
		assert.Equal(t, address.Of(int64(1000)), securityContext.RunAsGroup)
		assert.Equal(t, address.Of(true), securityContext.RunAsNonRoot)
		assert.Equal(t, address.Of(true), securityContext.Privileged)
		assert.Empty(t, securityContext.Capabilities)
	})

	t.Run(`privileged security context when feature flag is enabled for classic fullstack`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureRunOneAgentContainerPrivileged: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Nil(t, securityContext.RunAsUser)
		assert.Nil(t, securityContext.RunAsGroup)
		assert.Nil(t, securityContext.RunAsNonRoot)
		assert.Equal(t, address.Of(true), securityContext.Privileged)
		assert.Empty(t, securityContext.Capabilities)
	})
}

func TestPodSpecServiceAccountName(t *testing.T) {
	t.Run("service account name is unprivileged by default", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{}
		builder := builderInfo{
			instance:       dynakube,
			hostInjectSpec: dynakube.Spec.OneAgent.HostMonitoring,
		}
		podSpec := builder.podSpec()

		assert.Equal(t, defaultUnprivilegedServiceAccountName, podSpec.ServiceAccountName)
	})
}

func TestOneAgentResources(t *testing.T) {
	t.Run("get empty resources if hostInjection spec is nil", func(t *testing.T) {
		builder := builderInfo{}
		resources := builder.oneAgentResource()

		assert.Equal(t, corev1.ResourceRequirements{}, resources)
	})
	t.Run("get resources if hostInjection spec is set", func(t *testing.T) {
		builder := builderInfo{
			hostInjectSpec: &dynatracev1beta1.HostInjectSpec{
				OneAgentResources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1),
					},
				},
			},
		}
		resources := builder.oneAgentResource()

		assert.Equal(t, corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1),
			},
		}, resources)
	})
}
