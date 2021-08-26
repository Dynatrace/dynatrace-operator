package daemonset

import (
	"fmt"
	"strings"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

const (
	testImage = "test-image"
)

func TestUseImmutableImage(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`if image is unset and useImmutableImage is false, default image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent:         dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, defaultOneAgentImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is false, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Image: testImage,
				},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, testImage, podSpecs.Containers[0].Image)
	})
	t.Run(`if image is set and useImmutableImage is true, set image is used`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{
					Image: testImage,
				},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
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
	t.Run(`if image is unset and useImmutableImage is true, image is based on api url`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:latest", strings.TrimPrefix(testURL, "https://")))

		instance.Spec.OneAgent.Version = testValue
		dsInfo = NewClassicFullStack(&instance, log, testClusterID)
		ds, err = dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs = ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:%s", strings.TrimPrefix(testURL, "https://"), testValue))
	})
}

func TestCustomPullSecret(t *testing.T) {
	log := logger.NewDTLogger()
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
			},
			CustomPullSecret: testName,
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
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
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
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

		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
					Resources: corev1.ResourceRequirements{
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
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
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

func TestServiceAccountName(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`has default values`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}

		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.Equal(t, defaultUnprivilegedServiceAccountName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:   true,
					UseUnprivilegedMode: pointer.BoolPtr(true),
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		dsInfo = NewClassicFullStack(&instance, log, testClusterID)
		ds, err = dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs = ds.Spec.Template.Spec
		assert.Equal(t, defaultUnprivilegedServiceAccountName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:   true,
					UseUnprivilegedMode: pointer.BoolPtr(false),
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		dsInfo = NewClassicFullStack(&instance, log, testClusterID)
		ds, err = dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs = ds.Spec.Template.Spec
		assert.Equal(t, defaultServiceAccountName, podSpecs.ServiceAccountName)
	})
	t.Run(`uses custom value`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:  true,
					ServiceAccountName: testName,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}
		dsInfo := NewClassicFullStack(&instance, log, testClusterID)
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.Equal(t, testName, podSpecs.ServiceAccountName)

		instance = dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL:   testURL,
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				ClassicFullStack: dynatracev1alpha1.FullStackSpec{
					UseImmutableImage:  true,
					ServiceAccountName: testName,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					UseImmutableImage: true,
				},
			},
		}

		dsInfo = NewClassicFullStack(&instance, log, testClusterID)
		ds, err = dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs = ds.Spec.Template.Spec
		assert.Equal(t, testName, podSpecs.ServiceAccountName)
	})
}
