package validation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

func TestMissingCSIDaemonSet(t *testing.T) {
	t.Run(`valid cloud-native dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 2, &dynatracev1beta1.DynaKube{
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
		assertAllowedResponseWithWarnings(t, 2, &dynatracev1beta1.DynaKube{
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

	t.Run(`valid none-readonly host-monitoring dynakube specs`, func(t *testing.T) {
		objectMeta := defaultDynakubeObjectMeta.DeepCopy()
		objectMeta.Annotations = map[string]string{
			dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent: "false",
		}
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: *objectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		})
	})

	t.Run(`valid default host-monitoring dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
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
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{},
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
