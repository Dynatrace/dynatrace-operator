package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testFeature = "test-feature"
	testName    = "test-name"
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
		Name:       capability.StatsdIngestPortName,
		Protocol:   corev1.ProtocolUDP,
		Port:       capability.StatsdIngestPort,
		TargetPort: intstr.FromString(capability.StatsdIngestTargetPort),
	}
	agHttpsPort := corev1.ServicePort{
		Name:       capability.HttpsServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       capability.HttpsServicePort,
		TargetPort: intstr.FromString(capability.HttpsServicePortName),
	}
	agHttpPort := corev1.ServicePort{
		Name:       capability.HttpServicePortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       capability.HttpServicePort,
		TargetPort: intstr.FromString(capability.HttpServicePortName),
	}

	t.Run("check service name and selector", func(t *testing.T) {
		instance := testCreateInstance()
		service := createService(instance, testFeature, capability.AgServicePorts{
			Webserver: true,
		})

		assert.NotNil(t, service)
		assert.Equal(t, instance.Name+"-"+testFeature, service.Name)
		assert.Equal(t, instance.Namespace, service.Namespace)

		serviceSpec := service.Spec
		assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
		assert.Equal(t, map[string]string{
			kubeobjects.AppCreatedByLabel: testName,
			kubeobjects.AppComponentLabel: statefulset.ActiveGateComponentName,
			kubeobjects.FeatureLabel:      testFeature,
			kubeobjects.AppNameLabel:      version.AppName,
			kubeobjects.AppVersionLabel:   version.Version,
		}, serviceSpec.Selector)
	})

	t.Run("check AG service if metrics ingest enabled, but not StatsD", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{
			Webserver: true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.True(t, desiredPorts.AtLeastOneEnabled())

		service := createService(instance, testFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort)
		assert.NotContains(t, ports, statsdPort)
	})

	t.Run("check AG service if metrics ingest and StatsD enabled", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{
			Webserver: true,
			Statsd:    true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, desiredPorts.Statsd)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.AtLeastOneEnabled())

		service := createService(instance, testFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})

	t.Run("check AG service if StatsD enabled, but not metrics ingest", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{
			Statsd: true,
		}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, true)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.AtLeastOneEnabled())

		service := createService(instance, testFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort)
		assert.Contains(t, ports, statsdPort)
	})

	t.Run("check AG service if StatsD and metrics ingest are disabled", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{}
		testSetCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		testSetCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.False(t, desiredPorts.AtLeastOneEnabled())

		service := createService(instance, testFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})
}

func TestBuildServiceNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceHostName(testName, testFeature)
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = buildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}
