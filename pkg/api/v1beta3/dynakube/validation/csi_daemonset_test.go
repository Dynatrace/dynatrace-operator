package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func TestMissingCSIDaemonSet(t *testing.T) {
	t.Run(`valid cloud-native dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`valid default application-monitoring dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
				},
			},
		})
	})

	t.Run(`valid application-monitoring via csi dynakube specs`, func(t *testing.T) {
		useCSIDriver := true
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						UseCSIDriver: useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`valid default host-monitoring dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					HostMonitoring: &dynakube.HostInjectSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid cloud-native dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		assertDenied(t,
			[]string{errorCSIRequired},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
					},
				},
			})
	})

	t.Run(`invalid default host-monitoring dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		assertDenied(t,
			[]string{errorCSIRequired},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{},
					},
				},
			})
	})

	t.Run(`invalid application-monitoring via csi dynakube specs`, func(t *testing.T) {
		// no daemonset ==> fail
		useCSIDriver := true
		assertDenied(t,
			[]string{errorCSIRequired},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							UseCSIDriver: useCSIDriver,
						},
					},
				},
			})
	})
}

func TestDisabledCSIForReadonlyCSIVolume(t *testing.T) {
	objectMeta := defaultDynakubeObjectMeta.DeepCopy()
	objectMeta.Annotations = map[string]string{
		dynakube.AnnotationFeatureReadOnlyCsiVolume: "true",
	}

	t.Run(`valid cloud-native dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: *objectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid dynakube specs, as csi is disabled`, func(t *testing.T) {
		useCSIDriver := false
		assertDenied(t,
			[]string{errorCSIEnabledRequired},
			&dynakube.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
							UseCSIDriver: useCSIDriver,
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})

	t.Run(`invalid dynakube specs, as csi is not supported for feature`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorCSIEnabledRequired},
			&dynakube.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ClassicFullStack: &dynakube.HostInjectSpec{},
					},
				},
			})
	})
}
