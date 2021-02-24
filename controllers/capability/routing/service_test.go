package routing

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testFeature = "test-module"
	testName    = "test-name"
)

func TestCreateService(t *testing.T) {
	instance := &v1alpha1.DynaKube{
		ObjectMeta: v1.ObjectMeta{Namespace: testNamespace, Name: testName},
	}
	service := createService(instance, testFeature)

	assert.NotNil(t, service)
	assert.Equal(t, instance.Name+"-"+testFeature+"-service", service.Name)
	assert.Equal(t, instance.Namespace, service.Namespace)

	serviceSpec := service.Spec
	assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
	assert.Equal(t, map[string]string{
		keyFeature: testFeature,
	}, serviceSpec.Selector)

	ports := serviceSpec.Ports
	assert.Contains(t, ports, corev1.ServicePort{
		Protocol:   corev1.ProtocolTCP,
		Port:       9999,
		TargetPort: intstr.FromInt(9999),
	})
}
