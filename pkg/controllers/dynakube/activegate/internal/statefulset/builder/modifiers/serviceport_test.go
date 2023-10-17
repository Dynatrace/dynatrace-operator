package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setServicePortUsage(dynakube *dynatracev1beta1.DynaKube, isUsed bool) {
	if isUsed {
		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1beta1.MetricsIngestCapability.DisplayName)
	}
}

func TestServicePortEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.NewMap())

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, false)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.NewMap())

		assert.False(t, mod.Enabled())
	})
}

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.NewMap())
		builder := createBuilderForTesting()
		expectedPorts := mod.getPorts()
		expectedEnv := mod.getEnvs()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		container := sts.Spec.Template.Spec.Containers[0]
		isSubset(t, expectedPorts, container.Ports)
		isSubset(t, expectedEnv, container.Env)
		assert.Equal(t, consts.HttpsServicePortName, container.ReadinessProbe.HTTPGet.Port.StrVal)
	})
}

func TestBuildDNSEntryPoint(t *testing.T) {
	t.Run("DNSEntryPoint for ActiveGate routing capability", func(t *testing.T) {
		dynakubeActiveGateCapability := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
					},
				},
			},
		}
		cap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_ACTIVEGATE_SERVICE_HOST):$(DYNAKUBE_ACTIVEGATE_SERVICE_PORT)/communication,https://dynakube-activegate.dynatrace:$(DYNAKUBE_ACTIVEGATE_SERVICE_PORT)/communication", dnsEntryPoint)
	})

	t.Run("DNSEntryPoint for ActiveGate k8s monitoring capability", func(t *testing.T) {
		dynakubeActiveGateCapability := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		}
		cap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_ACTIVEGATE_SERVICE_HOST):$(DYNAKUBE_ACTIVEGATE_SERVICE_PORT)/communication", dnsEntryPoint)
	})

	t.Run("DNSEntryPoint for ActiveGate routing+kubemon capabilities", func(t *testing.T) {
		dynakubeActiveGateCapability := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
						dynatracev1beta1.RoutingCapability.DisplayName,
					},
				},
			},
		}
		cap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_ACTIVEGATE_SERVICE_HOST):$(DYNAKUBE_ACTIVEGATE_SERVICE_PORT)/communication,https://dynakube-activegate.dynatrace:$(DYNAKUBE_ACTIVEGATE_SERVICE_PORT)/communication", dnsEntryPoint)
	})

	t.Run("DNSEntryPoint for deprecated routing ActiveGate", func(t *testing.T) {
		dynakubeRoutingActiveGate := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: true,
				},
			},
		}
		cap := capability.NewRoutingCapability(&dynakubeRoutingActiveGate)
		portModifier := NewServicePortModifier(dynakubeRoutingActiveGate, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_ROUTING_SERVICE_HOST):$(DYNAKUBE_ROUTING_SERVICE_PORT)/communication,https://dynakube-routing.dynatrace:$(DYNAKUBE_ROUTING_SERVICE_PORT)/communication", dnsEntryPoint)
	})

	t.Run("DNSEntryPoint for deprecated kubernetes monitoring ActiveGate", func(t *testing.T) {
		dynakubeKubeMonActiveGate := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				},
			},
		}
		cap := capability.NewKubeMonCapability(&dynakubeKubeMonActiveGate)
		portModifier := NewServicePortModifier(dynakubeKubeMonActiveGate, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_KUBEMON_SERVICE_HOST):$(DYNAKUBE_KUBEMON_SERVICE_PORT)/communication", dnsEntryPoint)
	})

	t.Run("DNSEntryPoint for Synthetic capability", func(t *testing.T) {
		dynakubeSyntheticCapability := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureSyntheticLocationEntityId: "test",
				},
			},
		}
		cap := capability.NewSyntheticCapability(&dynakubeSyntheticCapability)
		portModifier := NewServicePortModifier(dynakubeSyntheticCapability, cap, prioritymap.NewMap())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_SYNTHETIC_SERVICE_HOST):$(DYNAKUBE_SYNTHETIC_SERVICE_PORT)/communication", dnsEntryPoint)
	})
}

func TestBuildServiceHostNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceHostName("test-name", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = buildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}

func TestBuildServiceDomainNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceDomainName("test-name", "test-namespace", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "test-name-test-component-feature.test-namespace:$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---dynakube_string"
	testNamespace := "this_is---namespace_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "this---dynakube_string-SHOULD--_--PaRsEcORrEcTlY.this_is---namespace_string:$(THIS___DYNAKUBE_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = buildServiceDomainName(testStringName, testNamespace, testStringFeature)
	assert.Equal(t, expected, actual)
}
