package capability

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testComponentFeature = "test-component-feature"
	testNamespace        = "test-namespace"
	testName             = "test-name"
	testApiUrl           = "https://demo.dev.dynatracelabs.com/api"
)

func testCreateInstance() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynakube.DynaKubeSpec{
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
			labels.AppCreatedByLabel: testName,
			labels.AppComponentLabel: labels.ActiveGateComponentLabel,
			labels.AppNameLabel:      version.AppName,
			labels.AppVersionLabel:   version.Version,
		}
		assert.Equal(t, expectedLabels, service.Labels)

		expectedSelector := map[string]string{
			labels.AppCreatedByLabel: testName,
			labels.AppManagedByLabel: version.AppName,
			labels.AppNameLabel:      labels.ActiveGateComponentLabel,
		}
		serviceSpec := service.Spec
		assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
		assert.Equal(t, expectedSelector, serviceSpec.Selector)
	})

	t.Run("check AG service if metrics-ingest disabled", func(t *testing.T) {
		instance := testCreateInstance()
		activegate.SwitchCapability(instance, dynakube.RoutingCapability, true)

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort)
		assert.NotContains(t, ports, agHttpPort)
	})
	t.Run("check AG service if metrics-ingest enabled", func(t *testing.T) {
		instance := testCreateInstance()
		activegate.SwitchCapability(instance, dynakube.MetricsIngestCapability, true)

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort)
		assert.Contains(t, ports, agHttpPort)
	})
}
