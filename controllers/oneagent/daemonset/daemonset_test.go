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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, podSpecs.Containers[0].Image, fmt.Sprintf("%s/linux/oneagent:latest", strings.TrimPrefix(testURL, "https://")))

		instance.Spec.OneAgent.Version = testValue
		dsInfo = NewClassicFullStack(&instance, log, testClusterID, "", "")
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
	dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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

		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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

		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo = NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo = NewClassicFullStack(&instance, log, testClusterID, "", "")
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
		dsInfo := NewClassicFullStack(&instance, log, testClusterID, "", "")
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

		dsInfo = NewClassicFullStack(&instance, log, testClusterID, "", "")
		ds, err = dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs = ds.Spec.Template.Spec
		assert.Equal(t, testName, podSpecs.ServiceAccountName)
	})
}

func TestInfraMon_SecurityContext(t *testing.T) {
	log := logger.NewDTLogger()
	t.Run(`No user and group id when not in read only mode`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
					FullStackSpec: dynatracev1alpha1.FullStackSpec{
						Enabled: true,
					},
					ReadOnly: dynatracev1alpha1.ReadOnlySpec{},
				},
			},
		}
		dsInfo := NewInfraMonitoring(&instance, log, testClusterID, "", "")
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContextConstraints := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContextConstraints)
		assert.Nil(t, securityContextConstraints.RunAsUser)
		assert.Nil(t, securityContextConstraints.RunAsGroup)
		assert.Nil(t, securityContextConstraints.RunAsNonRoot)
	})
	t.Run(`User and group id set when read only mode is enabled`, func(t *testing.T) {
		instance := dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				OneAgent: dynatracev1alpha1.OneAgentSpec{},
				InfraMonitoring: dynatracev1alpha1.InfraMonitoringSpec{
					FullStackSpec: dynatracev1alpha1.FullStackSpec{
						Enabled: true,
					},
					ReadOnly: dynatracev1alpha1.ReadOnlySpec{
						Enabled: true,
					},
				},
			},
		}
		dsInfo := NewInfraMonitoring(&instance, log, testClusterID, "", "")
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContextConstraints := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContextConstraints)
		assert.Nil(t, securityContextConstraints.RunAsNonRoot)
		assert.NotNil(t, securityContextConstraints.RunAsUser)
		assert.NotNil(t, securityContextConstraints.RunAsGroup)
		assert.Equal(t, int64(1001), *securityContextConstraints.RunAsUser)
		assert.Equal(t, int64(1001), *securityContextConstraints.RunAsGroup)
	})
}

func TestKubernetesVersion(t *testing.T) {
	const zeroFloat = float64(0)

	assert.Equal(t, 1.14, (&builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "14",
	}).kubernetesVersion())

	assert.Equal(t, 0.14, (&builderInfo{
		majorKubernetesVersion: "0",
		minorKubernetesVersion: "14",
	}).kubernetesVersion())

	assert.Equal(t, 123.2345, (&builderInfo{
		majorKubernetesVersion: "123",
		minorKubernetesVersion: "2345",
	}).kubernetesVersion())

	assert.Equal(t, 1.20, (&builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "20+",
	}).kubernetesVersion())

	assert.Equal(t, 1.20, (&builderInfo{
		majorKubernetesVersion: "1+",
		minorKubernetesVersion: "20+",
	}).kubernetesVersion())

	assert.Equal(t, 1.20, (&builderInfo{
		majorKubernetesVersion: "1+",
		minorKubernetesVersion: "20",
	}).kubernetesVersion())

	assert.Equal(t, float64(1), (&builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "",
	}).kubernetesVersion())

	assert.Equal(t, 0.1, (&builderInfo{
		majorKubernetesVersion: "",
		minorKubernetesVersion: "1",
	}).kubernetesVersion())

	assert.Equal(t, 1.14, (&builderInfo{
		majorKubernetesVersion: "-1",
		minorKubernetesVersion: "-14",
	}).kubernetesVersion())

	assert.Equal(t, 1.14, (&builderInfo{
		majorKubernetesVersion: "-1",
		minorKubernetesVersion: "14",
	}).kubernetesVersion())

	assert.Equal(t, 1.14, (&builderInfo{
		majorKubernetesVersion: "1",
		minorKubernetesVersion: "-14",
	}).kubernetesVersion())

	assert.Equal(t, 0.14, (&builderInfo{
		majorKubernetesVersion: "",
		minorKubernetesVersion: "-14",
	}).kubernetesVersion())

	assert.Equal(t, zeroFloat, (&builderInfo{
		majorKubernetesVersion: "",
		minorKubernetesVersion: "",
	}).kubernetesVersion())
}
