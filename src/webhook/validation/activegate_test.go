package validation

import (
	"fmt"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestConflictingActiveGateConfiguration(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		/*
			assertAllowedResponseWithoutWarnings(t, &dynatracev1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					Routing: dynatracev1beta1.RoutingSpec{
						Enabled: true,
					},
					KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
						Enabled: true,
					},
				},
			})
		*/
		assertAllowedResponseWithWarnings(t, 1, &dynatracev1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1.ActiveGateSpec{
					Capabilities: []dynatracev1.CapabilityDisplayName{
						dynatracev1.RoutingCapability.DisplayName,
						dynatracev1.KubeMonCapability.DisplayName,
					},
				},
			},
		})

		assertAllowedResponseWithWarnings(t, 3, &dynatracev1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1.ActiveGateSpec{
					Capabilities: []dynatracev1.CapabilityDisplayName{
						dynatracev1.MetricsIngestCapability.DisplayName,
					},
				},
			},
		})
	})
	/*
		t.Run(`conflicting dynakube specs`, func(t *testing.T) {
			assertDeniedResponse(t,
				[]string{errorConflictingActiveGateSections},
				&dynatracev1.DynaKube{
					ObjectMeta: defaultDynakubeObjectMeta,
					Spec: dynatracev1.DynaKubeSpec{
						APIURL: testApiUrl,
						Routing: dynatracev1beta1.RoutingSpec{
							Enabled: true,
						},
						ActiveGate: dynatracev1.ActiveGateSpec{
							Capabilities: []dynatracev1.CapabilityDisplayName{
								dynatracev1.RoutingCapability.DisplayName,
							},
						},
					},
				})
		})
	*/
}

func TestDuplicateActiveGateCapabilities(t *testing.T) {
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, dynatracev1.RoutingCapability.DisplayName)},
			&dynatracev1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1.ActiveGateSpec{
						Capabilities: []dynatracev1.CapabilityDisplayName{
							dynatracev1.RoutingCapability.DisplayName,
							dynatracev1.RoutingCapability.DisplayName,
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
			&dynatracev1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1.ActiveGateSpec{
						Capabilities: []dynatracev1.CapabilityDisplayName{
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
			&dynatracev1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1.ActiveGateSpec{
						Capabilities: []dynatracev1.CapabilityDisplayName{
							dynatracev1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run(`no memory warning in activeGate mode with memory limit`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1.ActiveGateSpec{
						Capabilities: []dynatracev1.CapabilityDisplayName{
							dynatracev1.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynatracev1.CapabilityProperties{
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

func TestSyntheticMonitoring(t *testing.T) {
	syntheticless := []dynatracev1.CapabilityDisplayName{
		dynatracev1.MetricsIngestCapability.DisplayName,
		dynatracev1.KubeMonCapability.DisplayName,
	}
	meta := defaultDynakubeObjectMeta.DeepCopy()
	meta.Annotations = map[string]string{
		dynatracev1.AnnotationFeatureSyntheticLocationEntityId: "doctored",
	}

	t.Run("denied synthetic and activegate capabilities", func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorJoinedSyntheticActiveGateCapability, syntheticless)},
			&dynatracev1.DynaKube{
				ObjectMeta: *meta,
				Spec: dynatracev1.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynatracev1.ActiveGateSpec{
						Capabilities: syntheticless,
					},
				},
			})
	})
}
