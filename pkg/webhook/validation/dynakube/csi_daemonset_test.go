package dynakube

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
)

func TestMissingCSIDaemonSet(t *testing.T) {
	t.Run(`valid cloud-native dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`valid default application-monitoring dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		})
	})

	t.Run(`valid application-monitoring via csi dynakube specs`, func(t *testing.T) {
		useCSIDriver := true
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						UseCSIDriver: &useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`valid default host-monitoring dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid cloud-native dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		assertDeniedResponse(t,
			[]string{errorCSIRequired},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
					},
				},
			})
	})

	t.Run(`invalid default host-monitoring dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		assertDeniedResponse(t,
			[]string{errorCSIRequired},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{},
					},
				},
			})
	})

	t.Run(`invalid application-monitoring via csi dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		useCSIDriver := true
		assertDeniedResponse(t,
			[]string{errorCSIRequired},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
							UseCSIDriver: &useCSIDriver,
						},
					},
				},
			})
	})
}

func TestDisabledCSIForReadonlyCSIVolume(t *testing.T) {
	objectMeta := defaultDynakubeObjectMeta.DeepCopy()
	objectMeta.Annotations = map[string]string{
		dynatracev1beta1.AnnotationFeatureReadOnlyCsiVolume: "true",
	}

	t.Run(`valid cloud-native dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: *objectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid dynakube specs, as csi is disabled`, func(t *testing.T) {
		useCSIDriver := false
		assertDeniedResponse(t,
			[]string{errorCSIEnabledRequired},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
							UseCSIDriver: &useCSIDriver,
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid dynakube specs, as csi is not supported for feature`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorCSIEnabledRequired},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
					},
				},
			})
	})
}
