package builder

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildActiveGatePodSpecs(t *testing.T) {
	t.Run("BuildActiveGatePodSpecs", func(t *testing.T) {
		serviceAccountName := MonitoringServiceAccount
		image := "image"
		instance := &dynatracev1alpha1.ActiveGate{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
			},
		}
		instance.Spec = dynatracev1alpha1.ActiveGateSpec{
			BaseActiveGateSpec: dynatracev1alpha1.BaseActiveGateSpec{
				ServiceAccountName: serviceAccountName,
				Image:              image,
			},
		}
		specs := BuildActiveGatePodSpecs(instance, nil, "")
		activeGateSpec := &instance.Spec
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.Equal(t, serviceAccountName, specs.ServiceAccountName)
		assert.Equal(t,
			*resource.NewScaledQuantity(1, -1),
			activeGateSpec.Resources.Requests[corev1.ResourceCPU])
		assert.NotNil(t, specs.Affinity)

		container := specs.Containers[0]
		assert.Equal(t, ActivegateName, container.Name)
		assert.Equal(t, image, container.Image)
		assert.Equal(t, container.Resources, activeGateSpec.Resources)
		assert.NotEmpty(t, container.Env)
		assert.LessOrEqual(t, 4, len(container.Env))
		assert.NotEmpty(t, container.Args)
		assert.LessOrEqual(t, 4, len(container.Args))
	})
	t.Run("BuildActiveGatePodSpecs with tenant info", func(t *testing.T) {
		instance := &dynatracev1alpha1.ActiveGate{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
			},
		}
		instance.Spec = dynatracev1alpha1.ActiveGateSpec{}
		specs := BuildActiveGatePodSpecs(instance, &dtclient.TenantInfo{
			ID:                    "tenant-id",
			Token:                 "tenant-token",
			CommunicationEndpoint: "tenant-endpoint",
		}, "")
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.NotNil(t, specs)
		assert.Equal(t, 1, len(specs.Containers))
		assert.Equal(t, MonitoringServiceAccount, specs.ServiceAccountName)
		assert.NotNil(t, specs.Affinity)

		container := specs.Containers[0]
		assert.Equal(t, ActivegateName, container.Name)
		assert.Equal(t, ActivegateImage, container.Image)
		assert.NotEmpty(t, container.Env)
		assert.LessOrEqual(t, 4, len(container.Env))
		assert.NotEmpty(t, container.Args)
		assert.LessOrEqual(t, 4, len(container.Args))

		envs := container.Env
		dtTenantExists := false
		dtTenantTokenExists := false
		dtTenantCommunicationEndpointsExists := false
		for _, env := range envs {
			if env.Name == DtTenant {
				dtTenantExists = true
				assert.Equal(t, "tenant-id", env.Value)
			}
			if env.Name == DtToken {
				dtTenantTokenExists = true
				assert.Equal(t, "tenant-token", env.Value)
			}
			if env.Name == DtServer {
				dtTenantCommunicationEndpointsExists = true
				assert.Equal(t, "tenant-endpoint", env.Value)
			}
		}
		assert.True(t, dtTenantExists)
		assert.True(t, dtTenantTokenExists)
		assert.True(t, dtTenantCommunicationEndpointsExists)
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
	instance := dynatracev1alpha1.ActiveGate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "dynatrace",
		},
	}
	envVars := buildEnvVars(&instance, &dtclient.TenantInfo{}, "cluster")
	assert.NotEmpty(t, envVars)
	assert.LessOrEqual(t, 6, len(envVars))

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
