package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDuplicateActiveGateCapabilities(t *testing.T) {
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorDuplicateActiveGateCapability, dynakube.RoutingCapability.DisplayName)},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynakube.ActiveGateSpec{
						Capabilities: []dynakube.CapabilityDisplayName{
							dynakube.RoutingCapability.DisplayName,
							dynakube.RoutingCapability.DisplayName,
						},
					},
				},
			})
	})
}

func TestInvalidActiveGateCapabilities(t *testing.T) {
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorInvalidActiveGateCapability, "invalid-capability")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynakube.ActiveGateSpec{
						Capabilities: []dynakube.CapabilityDisplayName{
							"invalid-capability",
						},
					},
				},
			})
	})
}

func TestMissingActiveGateMemoryLimit(t *testing.T) {
	t.Run(`memory warning in activeGate mode`, func(t *testing.T) {
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynakube.ActiveGateSpec{
						Capabilities: []dynakube.CapabilityDisplayName{
							dynakube.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynakube.CapabilityProperties{
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			})
	})
	t.Run(`no memory warning in activeGate mode with memory limit`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					ActiveGate: dynakube.ActiveGateSpec{
						Capabilities: []dynakube.CapabilityDisplayName{
							dynakube.RoutingCapability.DisplayName,
						},
						CapabilityProperties: dynakube.CapabilityProperties{
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
