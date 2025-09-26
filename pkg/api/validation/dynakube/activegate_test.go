package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDuplicateActiveGateCapabilities(t *testing.T) {
	t.Run("conflicting dynakube specs", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, activegate.RoutingCapability.DisplayName)},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
							activegate.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestInvalidActiveGateCapabilities(t *testing.T) {
	t.Run("conflicting dynakube specs", func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							"invalid-capability",
						},
					},
				},
			})
	})
}

func TestMissingActiveGateMemoryLimit(t *testing.T) {
	t.Run("memory warning in activeGate mode", func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
						CapabilityProperties: activegate.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run("no memory warning in activeGate mode with memory limit", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
						CapabilityProperties: activegate.CapabilityProperties{
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

func TestActiveGatePVCSettings(t *testing.T) {
	t.Run("EphemeralVolume disabled and PVC specified", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						UseEphemeralVolume:  false,
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{},
					},
				},
			})
	})
	t.Run("EphemeralVolume enabled and no PVC specified", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					ActiveGate: activegate.Spec{
						UseEphemeralVolume: true,
					},
				},
			})
	})
	t.Run("EphemeralVolume enabled and PVC specified", func(t *testing.T) {
		assertDenied(t,
			[]string{errorActiveGateInvalidPVCConfiguration},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:     testAPIURL,
					Extensions: &extensions.Spec{PrometheusSpec: &extensions.PrometheusSpec{}},
					ActiveGate: activegate.Spec{
						UseEphemeralVolume:  true,
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{},
					},
				},
			})
	})
}
