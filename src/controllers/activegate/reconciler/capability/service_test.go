package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	statsdingest "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability/statsd-ingest"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/statefulset"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testFeature = "test-feature"
	testName    = "test-name"
)

func TestCreateService(t *testing.T) {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
		},
	}
	service := createService(instance, testFeature)

	assert.NotNil(t, service)
	assert.Equal(t, instance.Name+"-"+testFeature, service.Name)
	assert.Equal(t, instance.Namespace, service.Namespace)

	serviceSpec := service.Spec
	assert.Equal(t, corev1.ServiceTypeClusterIP, serviceSpec.Type)
	assert.Equal(t, map[string]string{
		statefulset.KeyActiveGate: testName,
		statefulset.KeyDynatrace:  statefulset.ValueActiveGate,
		statefulset.KeyFeature:    testFeature,
	}, serviceSpec.Selector)

	ports := serviceSpec.Ports
	assert.Contains(t, ports,
		corev1.ServicePort{
			Name:       consts.HttpsServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HttpsServicePort,
			TargetPort: intstr.FromString(consts.HttpsServicePortName),
		},
		corev1.ServicePort{
			Name:       consts.HttpServicePortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       consts.HttpServicePort,
			TargetPort: intstr.FromString(consts.HttpServicePortName),
		},
	)
	if instance.NeedsStatsd() {
		assert.Contains(t, ports, corev1.ServicePort{
			Name:       statsdingest.StatsdIngestPortName,
			Protocol:   corev1.ProtocolUDP,
			Port:       statsdingest.StatsdIngestPort,
			TargetPort: intstr.FromString(statsdingest.StatsdIngestTargetPort),
		})
	} else {
		assert.NotContains(t, ports, corev1.ServicePort{
			Name:       statsdingest.StatsdIngestPortName,
			Protocol:   corev1.ProtocolUDP,
			Port:       statsdingest.StatsdIngestPort,
			TargetPort: intstr.FromString(statsdingest.StatsdIngestTargetPort),
		})
	}
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
