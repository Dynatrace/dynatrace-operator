package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testComponentFeature = "test-component-feature"
	testName             = "test-name"
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
	statsdPort := corev1.ServicePort{
		Name:       statefulset.StatsdIngestPortName,
		Protocol:   corev1.ProtocolUDP,
		Port:       statefulset.StatsdIngestPort,
		TargetPort: intstr.FromString(statefulset.StatsdIngestTargetPort),
	}
	agHttpsPort := corev1.ServicePort{
		Name:       HttpsServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       HttpsServicePort,
		TargetPort: intstr.FromString(HttpsServicePortName),
	}
	agHttpPort := corev1.ServicePort{
		Name:       HttpServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       HttpServicePort,
		TargetPort: intstr.FromString(HttpServicePortName),
	}

	t.Run("check service name, labels and selector", func(t *testing.T) {
		instance := testCreateInstance()
		service := createService(instance, testComponentFeature, AgServicePorts{
			Webserver: true,
		})

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

	t.Run("check AG service if metrics ingest enabled, but not StatsD", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := AgServicePorts{
			Webserver: true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := createService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort)
		assert.NotContains(t, ports, statsdPort)
	})

	t.Run("check AG service if metrics ingest and StatsD enabled", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := AgServicePorts{
			Webserver: true,
			Statsd:    true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, desiredPorts.Statsd)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := createService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})

	t.Run("check AG service if StatsD enabled, but not metrics ingest", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := AgServicePorts{
			Statsd: true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, true)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := createService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort)
		assert.Contains(t, ports, statsdPort)
	})

	t.Run("check AG service if StatsD and metrics ingest are disabled", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := AgServicePorts{}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.False(t, desiredPorts.HasPorts())

		service := createService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})
}

func TestBuildServiceNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceHostName(testName, testComponentFeature)
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = buildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}
