package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sversion "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

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
					Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
					ActiveGate: activegate.Spec{
						UseEphemeralVolume:  true,
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{},
					},
				},
			})
	})
}

func newValidatorWithK8sVersion(t *testing.T, major, minor string) *Validator {
	t.Helper()
	fakeClientSet := fakeclientset.NewClientset()
	fakeDisc := fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.FakedServerVersion = &k8sversion.Info{
		Major: major,
		Minor: minor,
	}

	return &Validator{
		apiReader:      fake.NewClient(),
		versionChecker: fakeDisc,
		modules:        installconfig.GetModules(),
	}
}

func TestActiveGateRollingUpdateWithGivenK8sVersion(t *testing.T) {
	maxUnavailable := intstr.FromString("20%")
	rollingUpdate := &appsv1.RollingUpdateStatefulSetStrategy{MaxUnavailable: &maxUnavailable}

	withRollingUpdate := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{
						RollingUpdate: rollingUpdate,
					},
				},
			},
		}
	}

	t.Run("no warning when rollingUpdate is not set", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}
		v := newValidatorWithK8sVersion(t, "1", "34")
		result := activeGateRollingUpdateWithOldK8sVersion(t.Context(), v, dk)
		assert.Empty(t, result)
	})

	t.Run("warning when rollingUpdate is set and k8s version is 1.34", func(t *testing.T) {
		v := newValidatorWithK8sVersion(t, "1", "34")
		result := activeGateRollingUpdateWithOldK8sVersion(t.Context(), v, withRollingUpdate())
		assert.Equal(t, warningActiveGateRollingUpdateOldK8sVersion, result)
	})

	t.Run("no warning when rollingUpdate is set and k8s version is 1.35", func(t *testing.T) {
		v := newValidatorWithK8sVersion(t, "1", "35")
		result := activeGateRollingUpdateWithOldK8sVersion(t.Context(), v, withRollingUpdate())
		assert.Empty(t, result)
	})

	t.Run("no warning when versionChecker is nil", func(t *testing.T) {
		v := &Validator{
			apiReader: fake.NewClient(),
			modules:   installconfig.GetModules(),
		}
		result := activeGateRollingUpdateWithOldK8sVersion(t.Context(), v, withRollingUpdate())
		assert.Empty(t, result)
	})
}
