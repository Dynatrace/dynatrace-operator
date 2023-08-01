package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testComponentFeature = "test-component-feature"
	testNamespace        = "test-namespace"
	testName             = "test-name"
	testApiUrl           = "https://demo.dev.dynatracelabs.com/api"
)

func testCreateInstance() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
	}
}

func TestCreateService(t *testing.T) {
	agHttpsPort := corev1.ServicePort{
		Name:       consts.HttpsServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HttpsServicePort,
		TargetPort: intstr.FromString(consts.HttpsServicePortName),
	}
	agHttpPort := corev1.ServicePort{
		Name:       consts.HttpServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.HttpServicePort,
		TargetPort: intstr.FromString(consts.HttpServicePortName),
	}

	t.Run("check service name, labels and selector", func(t *testing.T) {
		instance := testCreateInstance()
		service := CreateService(instance, testComponentFeature)

		assert.NotNil(t, service)
		assert.Equal(t, instance.Name+"-"+testComponentFeature, service.Name)
		assert.Equal(t, instance.Namespace, service.Namespace)

		expectedLabels := map[string]string{
			kubeobjects.AppCreatedByLabel: testName,
			kubeobjects.AppComponentLabel: kubeobjects.ActiveGateComponentLabel,
			kubeobjects.AppNameLabel:      version.AppName,
			kubeobjects.AppVersionLabel:   version.Version,
		}
		assert.Equal(t, expectedLabels, service.Labels)

		expectedSelector := map[string]string{
			kubeobjects.AppCreatedByLabel: testName,
			kubeobjects.AppManagedByLabel: version.AppName,
			kubeobjects.AppNameLabel:      kubeobjects.ActiveGateComponentLabel,
		}
		serviceSpec := service.Spec
		assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
		assert.Equal(t, expectedSelector, serviceSpec.Selector)
	})

	t.Run("check AG service if metrics-ingest disabled", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.RoutingCapability, true)

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort)
		assert.NotContains(t, ports, agHttpPort)
	})
	t.Run("check AG service if metrics-ingest enabled", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort)
		assert.Contains(t, ports, agHttpPort)
	})
}
