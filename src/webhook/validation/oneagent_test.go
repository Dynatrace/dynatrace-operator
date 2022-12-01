package validation

import (
	"fmt"
	"strconv"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
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
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
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
					HostMonitoring:   &dynatracev1beta1.HostInjectSpec{},
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
						ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
						HostMonitoring:   &dynatracev1beta1.HostInjectSpec{},
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
						HostMonitoring:        &dynatracev1beta1.HostInjectSpec{},
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
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
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
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
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
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`valid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertAllowedResponseWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "2"),
			&defaultCSIDaemonSet)

		assertAllowedResponseWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
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
						HostMonitoring: &dynatracev1beta1.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode:  "true",
				dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDeniedResponse(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynatracev1beta1.AnnotationFeatureMultipleOsAgentsOnNode: "false",
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
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
						UseCSIDriver: &useCSIDriver,
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
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
						UseCSIDriver: &useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})
}

func TestOneAgentVolumeStorageReadOnlyModeConflict(t *testing.T) {
	type conflictingSettingsTestcase struct {
		testName        string
		dynakubeFactory func(featureFlag string, featureFlagValue bool, oaEnvVar string, oaEnvValue bool) *dynatracev1beta1.DynaKube
		featureFlag     string
		featureValue    bool
	}

	oaVolumeStorageValue := false

	testcases := []conflictingSettingsTestcase{
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by feature flag in cloudNative",
			dynakubeFactory: createCloudNativeFullstackDynaKube,
			featureFlag:     dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:    true,
		},
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by feature flag in hostMonitoring",
			dynakubeFactory: createHostMonitoringDynaKube,
			featureFlag:     dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:    true,
		},
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by deprecated feature flag in cloudNative",
			dynakubeFactory: createCloudNativeFullstackDynaKube,
			featureFlag:     dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:    false,
		},
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by deprecated feature flag in hostMonitoring",
			dynakubeFactory: createHostMonitoringDynaKube,
			featureFlag:     dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:    false,
		},
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by default in cloudNative",
			dynakubeFactory: createCloudNativeFullstackDynaKube,
			featureFlag:     "",
			featureValue:    false,
		},
		{
			testName:        "disabled OneAgent volume storage by env var with OneAgent read only mode enabled by default in hostMonitoring",
			dynakubeFactory: createHostMonitoringDynaKube,
			featureFlag:     "",
			featureValue:    false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			assertDeniedResponse(t,
				[]string{errorVolumeStorageReadOnlyModeConflict},
				tc.dynakubeFactory(
					tc.featureFlag, tc.featureValue,
					oneagentEnableVolumeStorageEnvVarName, oaVolumeStorageValue),
				&defaultCSIDaemonSet)
		})
	}
}

type validOneAgentVolumeStorageReadOnlyModeSettingsTestcase struct {
	testName             string
	dynakubeFactory      func(featureFlag string, featureFlagValue bool, oaEnvVar string, oaEnvValue bool) *dynatracev1beta1.DynaKube
	featureFlag          string
	featureValue         bool
	oaVolumeStorageVar   string
	oaVolumeStorageValue bool
	acceptedWarningCount int
}

func TestOneAgentVolumeStorageReadOnlyModeNoConflict(t *testing.T) {
	testcases := []validOneAgentVolumeStorageReadOnlyModeSettingsTestcase{
		{
			testName:             "enabled OneAgent volume storage with OneAgent read only mode disabled by feature flag in cloudNative",
			dynakubeFactory:      createCloudNativeFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 0,
		},
		{
			testName:             "enabled OneAgent volume storage with OneAgent read only mode disabled by feature flag in hostMonitoring",
			dynakubeFactory:      createHostMonitoringDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 0,
		},
		{
			testName:             "enabled OneAgent volume storage with OneAgent read only mode disabled by deprecated feature flag in cloudNative",
			dynakubeFactory:      createCloudNativeFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 1,
		},
		{
			testName:             "enabled OneAgent volume storage with OneAgent read only mode disabled by deprecated feature flag in hostMonitoring",
			dynakubeFactory:      createHostMonitoringDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 1,
		},
		{
			testName:             "OneAgent volume storage default with OneAgent read only mode enabled by feature flag in cloudNative",
			dynakubeFactory:      createCloudNativeFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   "",
			acceptedWarningCount: 0,
		},
		{
			testName:             "OneAgent volume storage default with OneAgent read only mode enabled by feature flag in hostMonitoring",
			dynakubeFactory:      createHostMonitoringDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   "",
			acceptedWarningCount: 0,
		},
		{
			testName:             "OneAgent volume storage default with OneAgent read only mode enabled by default in cloudNative",
			dynakubeFactory:      createCloudNativeFullstackDynaKube,
			featureFlag:          "",
			oaVolumeStorageVar:   "",
			acceptedWarningCount: 0,
		},
		{
			testName:             "OneAgent volume storage default with OneAgent read only mode enabled by default in hostMonitoring",
			dynakubeFactory:      createHostMonitoringDynaKube,
			featureFlag:          "",
			oaVolumeStorageVar:   "",
			acceptedWarningCount: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			assertAllowedResponseWithWarnings(t,
				tc.acceptedWarningCount,
				tc.dynakubeFactory(tc.featureFlag, tc.featureValue, tc.oaVolumeStorageVar, tc.oaVolumeStorageValue),
				&defaultCSIDaemonSet)
		})
	}
}

func TestReadOnlyModeClassicFullstackWarning(t *testing.T) {
	testcases := []validOneAgentVolumeStorageReadOnlyModeSettingsTestcase{
		{
			testName:             "OneAgent volume storage disabled by env var and OneAgent read-only mode enabled by feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: false,
			acceptedWarningCount: 1,
		},
		{
			testName:             "OneAgent volume storage disabled by env var and OneAgent read-only mode enabled by deprecated feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: false,
			acceptedWarningCount: 2,
		},
		{
			testName:             "OneAgent volume storage disabled by env var and OneAgent read-only mode disabled by feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: false,
			acceptedWarningCount: 1,
		},
		{
			testName:             "OneAgent volume storage disabled by env var and OneAgent read-only mode disabled by deprecated feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: false,
			acceptedWarningCount: 2,
		},
		{
			testName:             "OneAgent volume storage disabled by env var and OneAgent read-only mode disabled by default in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          "",
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: false,
			acceptedWarningCount: 0,
		},
		{
			testName:             "OneAgent volume storage enabled by env var and OneAgent read-only mode enabled by feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 1,
		},
		{
			testName:             "OneAgent volume storage enabled by env var and OneAgent read-only mode enabled by deprecated feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 2,
		},
		{
			testName:             "OneAgent volume storage enabled by env var and OneAgent read-only mode disabled by feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureReadOnlyOneAgent,
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 1,
		},
		{
			testName:             "OneAgent volume storage enabled by env var and OneAgent read-only mode disabled by deprecated feature flag in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          dynatracev1beta1.AnnotationFeatureDisableReadOnlyOneAgent,
			featureValue:         true,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 2,
		},
		{
			testName:             "OneAgent volume storage enabled by env var and OneAgent read-only mode disabled by default in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          "",
			featureValue:         false,
			oaVolumeStorageVar:   oneagentEnableVolumeStorageEnvVarName,
			oaVolumeStorageValue: true,
			acceptedWarningCount: 0,
		},
		{
			testName:             "OneAgent volume storage default and OneAgent read-only mode disabled by default in classicFullstack",
			dynakubeFactory:      createClassicFullstackDynaKube,
			featureFlag:          "",
			featureValue:         false,
			oaVolumeStorageVar:   "",
			oaVolumeStorageValue: true,
			acceptedWarningCount: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			assertAllowedResponseWithWarnings(t,
				tc.acceptedWarningCount,
				tc.dynakubeFactory(tc.featureFlag, tc.featureValue, tc.oaVolumeStorageVar, tc.oaVolumeStorageValue),
				&defaultCSIDaemonSet)
		})
	}
}

func createHostInjectSpecWithOneAgentVolumeStorage(variable string, flag bool) *dynatracev1beta1.HostInjectSpec {
	his := &dynatracev1beta1.HostInjectSpec{}

	if len(variable) > 0 {
		his.Env = []corev1.EnvVar{
			{
				Name:  variable,
				Value: strconv.FormatBool(flag),
			},
		}
	}

	return his
}

func createFeatureFlaggedMetadata(featureFlag string, featureFlagValue bool) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
		Name:      "dynakube1",
		Namespace: testNamespace,
	}
	if len(featureFlag) > 0 {
		meta.Annotations = map[string]string{
			featureFlag: strconv.FormatBool(featureFlagValue),
		}
	}
	return meta
}

func createCloudNativeFullstackDynaKube(featureFlag string, featureFlagValue bool, oneAgentVolumeStorageVar string, oaEnvValue bool) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: createFeatureFlaggedMetadata(featureFlag, featureFlagValue),
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					HostInjectSpec: *createHostInjectSpecWithOneAgentVolumeStorage(oneAgentVolumeStorageVar, oaEnvValue),
				},
			},
		},
	}
}

func createHostMonitoringDynaKube(featureFlag string, featureFlagValue bool, oneAgentVolumeStorageVar string, oaEnvValue bool) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: createFeatureFlaggedMetadata(featureFlag, featureFlagValue),
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				HostMonitoring: createHostInjectSpecWithOneAgentVolumeStorage(oneAgentVolumeStorageVar, oaEnvValue),
			},
		},
	}
}

func createClassicFullstackDynaKube(featureFlag string, featureFlagValue bool, oneAgentVolumeStorageVar string, oaEnvValue bool) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: createFeatureFlaggedMetadata(featureFlag, featureFlagValue),
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: createHostInjectSpecWithOneAgentVolumeStorage(oneAgentVolumeStorageVar, oaEnvValue),
			},
		},
	}
}
