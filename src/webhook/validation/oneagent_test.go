package validation

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingOneAgentConfiguration(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{errorConflictingOneagentMode},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{},
						HostMonitoring:   &dynatracev1beta1.HostMonitoringSpec{},
					},
				},
			}, &defaultCSIDaemonSet)

		assertDeniedResponse(t,
			[]string{errorConflictingOneagentMode},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
						HostMonitoring:        &dynatracev1beta1.HostMonitoringSpec{},
					},
				},
			}, &defaultCSIDaemonSet)
	})
}

func TestConflictingNodeSelector(t *testing.T) {
	newCloudNativeDynakube := func(name string, annotations map[string]string, nodeSelectorValue string) *dynatracev1beta1.DynaKube {
		return &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   testNamespace,
				Annotations: annotations,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": nodeSelectorValue,
							},
						},
					},
				},
			},
		}
	}

	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict1",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)

		assertAllowedResponseWithoutWarnings(t,
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict2",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "2",
								},
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`valid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "2"),
			&defaultCSIDaemonSet)

		assertAllowedResponseWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs`, func(t *testing.T) {
		assertDeniedResponse(t,
			[]string{fmt.Sprintf(errorNodeSelectorConflict, "conflicting-dk")},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflicting-dk",
					Namespace: testNamespace,
				},
				Spec: dynatracev1beta1.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynatracev1beta1.OneAgentSpec{
						HostMonitoring: &dynatracev1beta1.HostMonitoringSpec{
							HostInjectSpec: dynatracev1beta1.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
				dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent:      "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureEnableMultipleOsAgentsOnNode: "false",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{}, "1"),
			&defaultCSIDaemonSet)
	})
}

func TestImageFieldSetWithoutCSIFlag(t *testing.T) {
	t.Run(`spec with appMon enabled and image name`, func(t *testing.T) {
		useCSIDriver := true
		testImage := "testImage"
		assertAllowedResponseWithoutWarnings(t, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						CodeModuleImage: testImage,
						UseCSIDriver:    &useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`spec with appMon enabled, useCSIDriver not enabled but image set`, func(t *testing.T) {
		useCSIDriver := false
		testImage := "testImage"
		assertDeniedResponse(t, []string{errorImageFieldSetWithoutCSIFlag}, &dynatracev1beta1.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						CodeModuleImage: testImage,
						UseCSIDriver:    &useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})

}
