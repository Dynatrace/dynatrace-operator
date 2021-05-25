package service

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/statefulset"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testFeature   = "test-feature"
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestCreateService(t *testing.T) {
	instance := &v1alpha1.DynaKube{
		ObjectMeta: v1.ObjectMeta{Namespace: testNamespace, Name: testName},
	}
	service := CreateService(instance, testFeature)

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
	assert.Contains(t, ports, corev1.ServicePort{
		Protocol:   corev1.ProtocolTCP,
		Port:       consts.ServicePort,
		TargetPort: intstr.FromString(consts.ServiceTargetPort),
	})
}

func TestBuildServiceNameForDNSEntryPoint(t *testing.T) {
	actual := BuildServiceHostName(testName, testFeature)
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = BuildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}
