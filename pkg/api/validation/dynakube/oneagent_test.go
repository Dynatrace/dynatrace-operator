package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingOneAgentConfiguration(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: &dynakube.HostInjectSpec{},
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ClassicFullStack: nil,
					HostMonitoring:   &dynakube.HostInjectSpec{},
				},
			},
		}, &defaultCSIDaemonSet)
	})
	t.Run(`conflicting dynakube specs`, func(t *testing.T) {
		assertDenied(t,
			[]string{errorConflictingOneagentMode},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ClassicFullStack: &dynakube.HostInjectSpec{},
						HostMonitoring:   &dynakube.HostInjectSpec{},
					},
				},
			}, &defaultCSIDaemonSet)

		assertDenied(t,
			[]string{errorConflictingOneagentMode},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
						HostMonitoring:        &dynakube.HostInjectSpec{},
					},
				},
			}, &defaultCSIDaemonSet)
	})
}

func TestConflictingNodeSelector(t *testing.T) {
	newCloudNativeDynakube := func(name string, annotations map[string]string, nodeSelectorValue string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   testNamespace,
				Annotations: annotations,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
						HostInjectSpec: dynakube.HostInjectSpec{
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
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict1",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)

		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict2",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
							HostInjectSpec: dynakube.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)

		assertAllowedWithoutWarnings(t, newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: logmonitoring.TemplateSpec{
							NodeSelector: map[string]string{"node": "12"},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`valid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertAllowedWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "2"),
			&defaultCSIDaemonSet)

		assertAllowedWithWarnings(t, 0,
			newCloudNativeDynakube("dk1", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs`, func(t *testing.T) {
		assertDenied(t,
			[]string{fmt.Sprintf(errorNodeSelectorConflict, "conflicting-dk")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
							HostInjectSpec: dynakube.HostInjectSpec{
								NodeSelector: map[string]string{
									"node": "1",
								},
							},
						},
					},
				},
			},
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflicting-dk",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs with multitenant hostMonitoring`, func(t *testing.T) {
		assertDenied(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "true",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDenied(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "false",
			}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{
				dynakube.AnnotationFeatureMultipleOsAgentsOnNode: "false",
			}, "1"),
			&defaultCSIDaemonSet)

		assertDenied(t, nil,
			newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			newCloudNativeDynakube("dk2", map[string]string{}, "1"),
			&defaultCSIDaemonSet)
	})
	t.Run(`invalid dynakube specs with existing log module`, func(t *testing.T) {
		assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, testName)},
			newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
				},
			}, &defaultCSIDaemonSet)

		assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, ""), testName, "dk2"},
			newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
				},
			},
			newCloudNativeDynakube("dk2", map[string]string{}, "1"),
			&defaultCSIDaemonSet)

		assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, testName)},
			newCloudNativeDynakube("dk1", map[string]string{}, "1"),
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL:        testApiUrl,
					LogMonitoring: &logmonitoring.Spec{},
					Templates: dynakube.TemplatesSpec{
						LogMonitoring: logmonitoring.TemplateSpec{
							NodeSelector: map[string]string{"node": "1"},
						},
					},
				},
			}, &defaultCSIDaemonSet)
	})
}

func TestImageFieldSetWithoutCSIFlag(t *testing.T) {
	t.Run(`spec with appMon enabled and image name`, func(t *testing.T) {
		useCSIDriver := true
		testImage := "testImage"
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						AppInjectionSpec: dynakube.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
						UseCSIDriver: useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})

	t.Run(`spec with appMon enabled, useCSIDriver not enabled but image set`, func(t *testing.T) {
		useCSIDriver := false
		testImage := "testImage"
		assertDenied(t, []string{errorImageFieldSetWithoutCSIFlag}, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						AppInjectionSpec: dynakube.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
						UseCSIDriver: useCSIDriver,
					},
				},
			},
		}, &defaultCSIDaemonSet)
	})
}

func createDynakube(oaEnvVar ...string) *dynakube.DynaKube {
	envVars := make([]corev1.EnvVar, 0)
	for i := 0; i < len(oaEnvVar); i += 2 {
		envVars = append(envVars, corev1.EnvVar{
			Name:  oaEnvVar[i],
			Value: oaEnvVar[i+1],
		})
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube",
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
					HostInjectSpec: dynakube.HostInjectSpec{
						Env: envVars,
					},
				},
			},
		},
	}
}

func TestUnsupportedOneAgentImage(t *testing.T) {
	type unsupportedOneAgentImageTests struct {
		testName        string
		envVars         []string
		allowedWarnings int
	}

	testcases := []unsupportedOneAgentImageTests{
		{
			testName:        "ONEAGENT_INSTALLER_SCRIPT_URL",
			envVars:         []string{"ONEAGENT_INSTALLER_SCRIPT_URL", "foobar"},
			allowedWarnings: 1,
		},
		{
			testName:        "ONEAGENT_INSTALLER_TOKEN",
			envVars:         []string{"ONEAGENT_INSTALLER_TOKEN", "foobar"},
			allowedWarnings: 1,
		},
		{
			testName:        "ONEAGENT_INSTALLER_SCRIPT_URL",
			envVars:         []string{"ONEAGENT_INSTALLER_SCRIPT_URL", "foobar", "ONEAGENT_INSTALLER_TOKEN", "foobar"},
			allowedWarnings: 1,
		},
		{
			testName:        "no env vars",
			envVars:         []string{},
			allowedWarnings: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			assertAllowedWithWarnings(t,
				tc.allowedWarnings,
				createDynakube(tc.envVars...),
				&defaultCSIDaemonSet)
		})
	}
}

func TestOneAgentHostGroup(t *testing.T) {
	t.Run(`valid dynakube specs`, func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{}, ""),
			&defaultCSIDaemonSet)

		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{"--other-param=1"}, ""),
			&defaultCSIDaemonSet)

		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{}, "field"),
			&defaultCSIDaemonSet)
	})

	t.Run(`obsolete settings`, func(t *testing.T) {
		assertAllowedWithWarnings(t,
			1,
			createDynakubeWithHostGroup([]string{"--set-host-group=arg"}, ""),
			&defaultCSIDaemonSet)

		assertAllowedWithWarnings(t,
			1,
			createDynakubeWithHostGroup([]string{"--set-host-group=arg"}, "field"),
			&defaultCSIDaemonSet)

		assertAllowedWithWarnings(t,
			1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ClassicFullStack: &dynakube.HostInjectSpec{
							Args: []string{"--set-host-group=arg"},
						},
						HostGroup: "",
					},
				},
			},
			&defaultCSIDaemonSet)

		assertAllowedWithWarnings(t,
			1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						HostMonitoring: &dynakube.HostInjectSpec{
							Args: []string{"--set-host-group=arg"},
						},
						HostGroup: "",
					},
				},
			},
			&defaultCSIDaemonSet)
	})
}

func createDynakubeWithHostGroup(args []string, hostGroup string) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: defaultDynakubeObjectMeta,
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{
					HostInjectSpec: dynakube.HostInjectSpec{
						Args: args,
					},
				},
				HostGroup: hostGroup,
			},
		},
	}
}

func TestIsOneAgentVersionValid(t *testing.T) {
	dk := dynakube.DynaKube{
		ObjectMeta: defaultDynakubeObjectMeta,
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: dynakube.OneAgentSpec{
				ClassicFullStack: &dynakube.HostInjectSpec{},
			},
		},
	}

	validVersions := []string{"", "1.0.0.20240101-000000"}
	invalidVersions := []string{
		"latest",
		"raw",
		"1.200.1-raw",
		"v1.200.1-raw",
		"1.200.1+build",
		"v1.200.1+build",
		"1.200.1-raw+build",
		"v1.200.1-raw+build",
		"1.200",
		"1.200.0",
		"1.200.0.0",
		"1.200.0.0-0",
		"v1.200",
		"1",
		"v1",
		"1.0",
		"v1.0",
		"v1.200.0",
	}

	for _, validVersion := range validVersions {
		dk.Spec.OneAgent.ClassicFullStack.Version = validVersion
		t.Run(fmt.Sprintf("OneAgent custom version %s is allowed", validVersion), func(t *testing.T) {
			assertAllowed(t, &dk)
		})
	}

	for _, invalidVersion := range invalidVersions {
		dk.Spec.OneAgent.ClassicFullStack.Version = invalidVersion
		t.Run(fmt.Sprintf("OneAgent custom version %s is not allowed", invalidVersion), func(t *testing.T) {
			assertDenied(t, []string{versionInvalidMessage}, &dk)
		})
	}
}
