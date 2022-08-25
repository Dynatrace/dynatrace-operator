package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
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
		Name:       consts.StatsdIngestPortName,
		Protocol:   corev1.ProtocolUDP,
		Port:       consts.StatsdIngestPort,
		TargetPort: intstr.FromString(consts.StatsdIngestTargetPort),
	}
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
		service := CreateService(instance, testComponentFeature, capability.AgServicePorts{
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
		desiredPorts := capability.AgServicePorts{
			Webserver: true,
		}
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := CreateService(instance, testComponentFeature, desiredPorts)
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
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, desiredPorts.Statsd)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := CreateService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})

	t.Run("check AG service if StatsD enabled, but not metrics ingest", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{
			Statsd: true,
		}
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, true)
		require.True(t, instance.NeedsStatsd())
		require.True(t, desiredPorts.HasPorts())

		service := CreateService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort)
		assert.Contains(t, ports, statsdPort)
	})

	t.Run("check AG service if StatsD and metrics ingest are disabled", func(t *testing.T) {
		instance := testCreateInstance()
		desiredPorts := capability.AgServicePorts{}
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.NeedsStatsd())
		require.False(t, desiredPorts.HasPorts())

		service := CreateService(instance, testComponentFeature, desiredPorts)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})
}

func TestBuildServiceNameForDNSEntryPoint(t *testing.T) {
	actual := BuildServiceHostName(testName, testComponentFeature)
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = BuildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}
