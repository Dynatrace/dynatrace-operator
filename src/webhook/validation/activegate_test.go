package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestConflictingActiveGateConfiguration(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {

		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				Routing: dynatracev1beta1.RoutingSpec{
					Enabled: true,
				},
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				},
			},
		})

		assertAllowedResponseWithWarnings(t, 1, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.RoutingCapability.DisplayName,
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
		})

		assertAllowedResponseWithWarnings(t, 3, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.MetricsIngestCapability.DisplayName,
					},
				},
			},
		})
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorConflictingActiveGateSections},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					Routing: dynatracev1beta1.RoutingSpec{
						Enabled: true,
					},
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestDuplicateActiveGateCapabilities(t *testing.T) {

	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1beta1.RoutingCapability.DisplayName)},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
							dynatracev1beta1.RoutingCapability.DisplayName,
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
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
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
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta1.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run(`no memory warning in activeGate mode with memory limit`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1beta1.ActiveGateSpec{
						Capabilities: []dynatracev1beta1.CapabilityDisplayName{
							dynatracev1beta1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1beta1.CapabilityProperties{
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
