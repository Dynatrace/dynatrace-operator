package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	testImage = "test-image"
)

func TestUseImmutableImage(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`if image is unset, image`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
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
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
						Image: testImage,
					},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
}

func TestCustomPullSecret(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testURL,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
			},
			CustomPullSecret: testName,
		},
	}
	dsInfo := NewClassicFullStack(&instance, log, testClusterID)
	ds, err := dsInfo.BuildDaemonSet()
	require.NoError(t, err)

	podSpecs := ds.Spec.Template.Spec
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.ImagePullSecrets)
	assert.Equal(t, testName, podSpecs.ImagePullSecrets[0].Name)
}

func TestResources(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`minimal cpu request of 100mC is set if no resources specified`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
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
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
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
			},
		}

		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
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

func TestInfraMon_SecurityContext(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`User and group id set when read only mode is enabled`, func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}
		dsInfo := NewHostMonitoring(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContextConstraints := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContextConstraints)
		assert.Nil(t, securityContextConstraints.RunAsNonRoot)
	})
}
