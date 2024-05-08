package dynakube

import (
	"fmt"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDuplicateActiveGateCapabilities(t *testing.T) {
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1beta2.RoutingCapability.DisplayName)},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestInvalidActiveGateCapabilities(t *testing.T) {
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability")},
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							"invalid-capability",
						},
					},
				},
			})
	})
}

func TestMissingActiveGateMemoryLimit(t *testing.T) {
	t.Run(`memory warning in activeGate mode`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 1,
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta2.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run(`no memory warning in activeGate mode with memory limit`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta2.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta2.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta2.CapabilityProperties{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceLimitsMemory: *resource.NewMilliQuantity(1, ""),
								},
							},
						},
					},
				},
			})
	})
}
