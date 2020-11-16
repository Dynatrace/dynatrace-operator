package builder

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildActiveGatePodSpecs(t *testing.T) {
	t.Run("BuildActiveGatePodSpecs", func(t *testing.T) {
		serviceAccountName := MonitoringServiceAccount
		image := "test-url.com/linux/activegates"
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
			},
		}
		instance.Spec = dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://test-url.com/api",
			KubernetesMonitoringSpec: dynatracev1alpha1.KubernetesMonitoringSpec{
				ServiceAccountName: serviceAccountName,
				Image:              image,
			},
		}
		specs, err := BuildActiveGatePodSpecs(instance, "")
		assert.NoError(t, err)
		activeGateSpec := &instance.Spec
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.Equal(t, serviceAccountName, specs.ServiceAccountName)
		assert.True(t,
			activeGateSpec.KubernetesMonitoringSpec.Resources.Requests[corev1.ResourceCPU].Equal(
				*resource.NewScaledQuantity(150, resource.Milli)))
		assert.True(t,
			activeGateSpec.KubernetesMonitoringSpec.Resources.Requests[corev1.ResourceMemory].Equal(
				*resource.NewScaledQuantity(250, resource.Mega)))
		assert.True(t,
			activeGateSpec.KubernetesMonitoringSpec.Resources.Limits[corev1.ResourceCPU].Equal(
				*resource.NewScaledQuantity(300, resource.Milli)))
		assert.True(t,
			activeGateSpec.KubernetesMonitoringSpec.Resources.Limits[corev1.ResourceMemory].Equal(
				*resource.NewScaledQuantity(1, resource.Giga)))
		assert.NotNil(t, specs.Affinity)

		container := specs.Containers[0]
		assert.Equal(t, ActivegateName, container.Name)
		assert.Equal(t, image, container.Image)
		assert.Equal(t, container.Resources, activeGateSpec.KubernetesMonitoringSpec.Resources)
		assert.NotEmpty(t, container.Env)
		assert.LessOrEqual(t, 3, len(container.Env))
		assert.NotEmpty(t, container.Args)
		assert.LessOrEqual(t, 1, len(container.Args))
	})
	t.Run("BuildActiveGatePodSpecs with tenant info", func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
			},
		}
		instance.Spec = dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://test-env.com",
		}
		specs, err := BuildActiveGatePodSpecs(instance, "")
		assert.NoError(t, err)
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.Equal(t, MonitoringServiceAccount, specs.ServiceAccountName)
		assert.NotNil(t, specs.Affinity)

		container := specs.Containers[0]
		assert.Equal(t, ActivegateName, container.Name)
		assert.Equal(t, "test-env.com/linux/activegate", container.Image)
		assert.NotEmpty(t, container.Env)
		assert.LessOrEqual(t, 3, len(container.Env))
		assert.NotEmpty(t, container.Args)
		assert.LessOrEqual(t, 1, len(container.Args))
	})
}

func TestBuildLabels(t *testing.T) {
	t.Run("BuildLabels", func(t *testing.T) {
		someLables := make(map[string]string)
		someLables["label"] = "test"
		labels := BuildLabels("test-labels", someLables)
		assert.NotEmpty(t, labels)
		assert.Equal(t, "test", labels["label"])
		assert.Equal(t, "activegate", labels["dynatrace"])
		assert.Equal(t, "test-labels", labels["activegate"])
	})
	t.Run("BuildLabels handle nil value", func(t *testing.T) {
		labels := BuildLabels("test-labels", nil)
		assert.NotEmpty(t, labels)
		assert.Equal(t, "activegate", labels["dynatrace"])
		assert.Equal(t, "test-labels", labels["activegate"])
	})
}

func TestBuildEnvVars(t *testing.T) {
	instance := dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dynatrace",
		},
	}
	envVars := buildEnvVars(&instance, "cluster")
	assert.NotEmpty(t, envVars)
	assert.LessOrEqual(t, 3, len(envVars))

	hasNamespace := false
	hasClusterName := false

	for _, envVar := range envVars {
		if envVar.Name == DtIdSeedNamespace {
			assert.Equal(t, "dynatrace", envVar.Value)
			hasNamespace = true
		} else if envVar.Name == DtIdSeedClusterId {
			assert.Equal(t, "cluster", envVar.Value)
			hasClusterName = true
		}

	}

	assert.True(t, hasNamespace)
	assert.True(t, hasClusterName)
}
