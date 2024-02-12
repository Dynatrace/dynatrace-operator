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

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, false)
		multiCapability := capability.NewMultiCapability(&dynakube)

		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())

		assert.False(t, mod.Enabled())
	})
}

func TestServicePortModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		setServicePortUsage(&dynakube, true)
		multiCapability := capability.NewMultiCapability(&dynakube)
		mod := NewServicePortModifier(dynakube, multiCapability, prioritymap.New())
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
		multiCap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, multiCap, prioritymap.New())
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
		multiCap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, multiCap, prioritymap.New())
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
		multiCap := capability.NewMultiCapability(&dynakubeActiveGateCapability)
		portModifier := NewServicePortModifier(dynakubeActiveGateCapability, multiCap, prioritymap.New())
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
		multiCap := capability.NewRoutingCapability(&dynakubeRoutingActiveGate)
		portModifier := NewServicePortModifier(dynakubeRoutingActiveGate, multiCap, prioritymap.New())
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
		multiCap := capability.NewKubeMonCapability(&dynakubeKubeMonActiveGate)
		portModifier := NewServicePortModifier(dynakubeKubeMonActiveGate, multiCap, prioritymap.New())
		dnsEntryPoint := portModifier.buildDNSEntryPoint()
		assert.Equal(t, "https://$(DYNAKUBE_KUBEMON_SERVICE_HOST):$(DYNAKUBE_KUBEMON_SERVICE_PORT)/communication", dnsEntryPoint)
	})
}
