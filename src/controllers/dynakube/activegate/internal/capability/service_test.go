package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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
	testNamespace        = "test-namespace"
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

	t.Run("check AG service if metrics ingest enabled, but not StatsD", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.IsStatsdCapabilityEnabled())

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort)
		assert.NotContains(t, ports, statsdPort)
	})

	t.Run("check AG service if metrics ingest and StatsD enabled", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, true)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, true)
		require.True(t, instance.IsStatsdCapabilityEnabled())

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.Contains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})

	t.Run("check AG service if StatsD enabled, but not metrics ingest", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, true)
		require.True(t, instance.IsStatsdCapabilityEnabled())

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort)
		assert.Contains(t, ports, statsdPort)
	})

	t.Run("check AG service if StatsD and metrics ingest are disabled", func(t *testing.T) {
		instance := testCreateInstance()
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.MetricsIngestCapability, false)
		kubeobjects.SwitchCapability(instance, dynatracev1beta1.StatsdIngestCapability, false)
		require.True(t, !instance.IsStatsdCapabilityEnabled())

		service := CreateService(instance, testComponentFeature)
		ports := service.Spec.Ports

		assert.NotContains(t, ports, agHttpsPort, agHttpPort, statsdPort)
	})
}
