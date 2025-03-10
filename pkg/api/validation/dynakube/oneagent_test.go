package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConflictingOneAgentConfiguration(t *testing.T) {
	t.Run("valid dynakube specs", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ClassicFullStack: nil,
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
					HostMonitoring:   nil,
				},
			},
		})

		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ClassicFullStack: nil,
					HostMonitoring:   &oneagent.HostInjectSpec{},
				},
			},
		})
	})
	t.Run("conflicting dynakube specs", func(t *testing.T) {
		assertDenied(t,
			[]string{errorConflictingOneagentMode},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						ClassicFullStack: &oneagent.HostInjectSpec{},
						HostMonitoring:   &oneagent.HostInjectSpec{},
					},
				},
			})

		assertDenied(t,
			[]string{errorConflictingOneagentMode},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						HostMonitoring:        &oneagent.HostInjectSpec{},
					},
				},
			})
	})
}

func TestConflictingNodeSelector(t *testing.T) {
	newCloudNativeDynakube := func(name, apiUrl, nodeSelectorValue string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: apiUrl,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": nodeSelectorValue,
							},
						},
					},
				},
			},
		}
	}

	t.Run("valid dynakube specs - 2 host-monitoring DK, different nodes", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
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
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
							},
						},
					},
				},
			})
	})
	t.Run("valid dynakube specs - 1 cloud-native + 1 host-monitoring DK, different nodes", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conflict2",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
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
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "2",
							},
						},
					},
				},
			})
	})

	t.Run("valid dynakube specs - 1 cloud-native + 1 log-monitoring DK, same tenant, different nodes", func(t *testing.T) {
		api1 := "https://f1.q.d.n/api"

		assertAllowedWithoutWarnings(t, newCloudNativeDynakube("dk1", api1, "1"),
			createStandaloneLogMonitoringDynakube("dk-lm", api1, "12"))
	})

	t.Run("valid dynakube specs - 1 cloud-native + 1 log-monitoring DK, different tenant, same nodes", func(t *testing.T) {
		api1 := "https://f1.q.d.n/api"
		api2 := "https://f2.q.d.n/api"
		assertAllowedWithoutWarnings(t, newCloudNativeDynakube("dk1", api1, "1"),
			createStandaloneLogMonitoringDynakube("dk-lm", api2, "1"))
	})

	t.Run("valid dynakube specs - 2 log-monitoring DK, different tenant, same nodes", func(t *testing.T) {
		api1 := "https://f1.q.d.n/api"
		api2 := "https://f2.q.d.n/api"
		assertAllowedWithWarnings(t, 1, createStandaloneLogMonitoringDynakube("dk1", api1, "1"),
			createStandaloneLogMonitoringDynakube("dk-lm", api2, "1"))
	})

	t.Run("invalid dynakube specs - 1 cloud-native + 1 host-monitoring DK, SAME nodes, different tenant", func(t *testing.T) {
		api1 := "https://f1.q.d.n/api"
		api2 := "https://f2.q.d.n/api"
		assertDenied(t,
			[]string{fmt.Sprintf(errorNodeSelectorConflict, "conflicting-dk")},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: api1,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
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
					APIURL: api2,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							NodeSelector: map[string]string{
								"node": "1",
							},
						},
					},
				},
			})
		t.Run("invalid dynakube specs - 1 cloud-native + 1 log-monitoring DK, same tenant, same nodes", func(t *testing.T) {
			api1 := "https://f1.q.d.n/api"

			assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, "dk-lm")},
				newCloudNativeDynakube("dk-cm", api1, "1"),
				createStandaloneLogMonitoringDynakube("dk-lm", api1, "1"))
		})
		t.Run("multiple invalid dynakube specs - 2 cloud-native + 1 log-monitoring DK, same tenant, same nodes", func(t *testing.T) {
			api1 := "https://f1.q.d.n/api"

			assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, ""), "dk-lm", "dk-cm2"},
				newCloudNativeDynakube("dk-cm1", api1, "1"),
				createStandaloneLogMonitoringDynakube("dk-lm", api1, ""),
				newCloudNativeDynakube("dk-cm2", api1, "1"))
		})

		t.Run("invalid dynakube specs - 1 log-monitoring DK + 1 cloud-native, same tenant, same nodes", func(t *testing.T) {
			api1 := "https://f1.q.d.n/api"

			assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, "dk-cn")},
				createStandaloneLogMonitoringDynakube("dk-lm", api1, "1"),
				newCloudNativeDynakube("dk-cn", api1, "1"))
		})

		t.Run("some invalid dynakube specs - 2 log-monitoring DK + 1 cloud-native, 2 tenants, same nodes", func(t *testing.T) {
			api1 := "https://f1.q.d.n/api"
			api2 := "https://f2.q.d.n/api"

			assertDenied(t, []string{fmt.Sprintf(errorNodeSelectorConflict, "dk-lm2")},
				createStandaloneLogMonitoringDynakube("dk-lm1", api1, "1"),
				newCloudNativeDynakube("dk-cm1", api2, "1"),
				createStandaloneLogMonitoringDynakube("dk-lm2", api1, "1"))
		})
	})
}

func setupDisabledCSIEnv(t *testing.T) {
	t.Helper()
	installconfig.SetModulesOverride(t, installconfig.Modules{
		CSIDriver:      false,
		ActiveGate:     true,
		OneAgent:       true,
		Extensions:     true,
		LogMonitoring:  true,
		EdgeConnect:    true,
		Supportability: true,
	})
}

func TestImageFieldSetWithoutCSIFlag(t *testing.T) {
	t.Run("spec with appMon enabled and image name", func(t *testing.T) {
		testImage := "testImage"
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
					},
				},
			},
		})
	})

	t.Run("spec with appMon enabled, csi driver not enabled but image set", func(t *testing.T) {
		setupDisabledCSIEnv(t)

		testImage := "testImage"
		assertDenied(t, []string{errorImageFieldSetWithoutCSIFlag}, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
					},
				},
			},
		})
	})

	t.Run("spec with appMon enabled, csi driver not enabled but remote image download enabled and image set", func(t *testing.T) {
		setupDisabledCSIEnv(t)

		testImage := "testImage"
		assertAllowed(t, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynakube.AnnotationFeatureRemoteImageDownload: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
					},
				},
			},
		})
	})

	t.Run("spec with appMon enabled, csi driver and remote image download not enabled and image set", func(t *testing.T) {
		setupDisabledCSIEnv(t)

		testImage := "testImage"
		assertDenied(t, []string{errorImageFieldSetWithoutCSIFlag}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynakube.AnnotationFeatureRemoteImageDownload: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
					},
				},
			},
		})
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
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
					HostInjectSpec: oneagent.HostInjectSpec{
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
				createDynakube(tc.envVars...))
		})
	}
}

func TestOneAgentHostGroup(t *testing.T) {
	t.Run("valid dynakube specs", func(t *testing.T) {
		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{}, ""))

		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{"--other-param=1"}, ""))

		assertAllowedWithoutWarnings(t,
			createDynakubeWithHostGroup([]string{}, "field"))
	})

	t.Run("obsolete settings", func(t *testing.T) {
		assertAllowedWithWarnings(t,
			1,
			createDynakubeWithHostGroup([]string{"--set-host-group=arg"}, ""))

		assertAllowedWithWarnings(t,
			1,
			createDynakubeWithHostGroup([]string{"--set-host-group=arg"}, "field"))

		assertAllowedWithWarnings(t,
			1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						ClassicFullStack: &oneagent.HostInjectSpec{
							Args: []string{"--set-host-group=arg"},
						},
						HostGroup: "",
					},
				},
			})

		assertAllowedWithWarnings(t,
			1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							Args: []string{"--set-host-group=arg"},
						},
						HostGroup: "",
					},
				},
			})
	})
}

func createDynakubeWithHostGroup(args []string, hostGroup string) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: defaultDynakubeObjectMeta,
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
					HostInjectSpec: oneagent.HostInjectSpec{
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
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
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
		dk.OneAgent().ClassicFullStack.Version = validVersion
		t.Run(fmt.Sprintf("OneAgent custom version %s is allowed", validVersion), func(t *testing.T) {
			assertAllowed(t, &dk)
		})
	}

	for _, invalidVersion := range invalidVersions {
		dk.OneAgent().ClassicFullStack.Version = invalidVersion
		t.Run(fmt.Sprintf("OneAgent custom version %s is not allowed", invalidVersion), func(t *testing.T) {
			assertDenied(t, []string{versionInvalidMessage}, &dk)
		})
	}
}

func TestPublicImageSetWithReadOnlyMode(t *testing.T) {
	t.Run("reject dk with hostMon without csi and custom image", func(t *testing.T) {
		setupDisabledCSIEnv(t)
		assertAllowedWithWarnings(t, 1,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							Image: "test/image/test-image:some-tag",
						},
					},
				},
			})
	})
	t.Run("allow dk with hostMon without csi and no custom image", func(t *testing.T) {
		setupDisabledCSIEnv(t)
		assertAllowedWithoutWarnings(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{},
					},
				},
			})
	})
	t.Run("allow dk with hostMon with csi and custom image", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{
						Image: "test/image/test-image:some-tag",
					},
				},
			},
		})
	})
	t.Run("allow dk with classicFullStack without csi and custom image", func(t *testing.T) {
		setupDisabledCSIEnv(t)
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: "test/image/test-image:some-tag",
					},
				},
			},
		})
	})
	t.Run("allow dk with classicFullStack with csi and custom image", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image: "test/image/test-image:some-tag",
					},
				},
			},
		})
	})
}

func TestOneAgentArguments(t *testing.T) {
	type oneAgentArgumentTest struct {
		testName      string
		arguments     []string
		expectedError string
	}

	testcases := []oneAgentArgumentTest{
		{
			testName: "duplicate arguments are rejected",
			arguments: []string{
				"--set-server=foo",
				"--set-server=bar",
			},
			expectedError: fmt.Sprintf(errorDuplicateOneAgentArgument, "--set-server"),
		},
		{
			testName: "duplicate arguments with same value are rejected",
			arguments: []string{
				"--set-server=foo",
				"--set-server=foo",
			},
			expectedError: fmt.Sprintf(errorDuplicateOneAgentArgument, "--set-server"),
		},
		{
			testName: "no duplicate arguments",
			arguments: []string{
				"--set-server=foo",
				"--set-host-source-id=bar",
			},
			expectedError: "",
		},
		{
			testName: "duplicate host property",
			arguments: []string{
				"--set-server=foo",
				"--set-host-property=foo1=bar1",
				"--set-host-property=foo2=bar2",
				"--set-host-property=foo3=bar3",
				"--set-host-property=foo3=bar3",
				"--set-host-property=foo2=bar2",
				"--set-host-property=foo1=bar1",
			},
			expectedError: "",
		},
		{
			testName: "duplicate host tag",
			arguments: []string{
				"--set-server=foo",
				"--set-host-tag=foo=1",
				"--set-host-tag=bar=1",
				"--set-host-tag=dow=1",
			},
			expectedError: "",
		},
		{
			testName: "duplicate host tag with same value",
			arguments: []string{
				"--set-host-tag=foo=1",
				"--set-host-tag=bar",
				"--set-host-tag=foo=1",
				"--set-host-tag=bar",
				"--set-host-tag=doh",
				"--set-host-tag=bar",
				"--set-host-tag=foo=1",
			},
			expectedError: fmt.Sprintf(errorSameHostTagMultipleTimes, "[foo=1 bar]"),
		},
		{
			testName: "arguments without value",
			arguments: []string{
				"--enable-feature-a",
				"--enable-feature-b",
				"--enable-feature-c",
			},
			expectedError: "",
		},
		{
			testName: "duplicate arguments without value",
			arguments: []string{
				"--enable-feature-a",
				"--enable-feature-b",
				"--enable-feature-a",
				"--enable-feature-c",
			},
			expectedError: fmt.Sprintf(errorDuplicateOneAgentArgument, "--enable-feature-a"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			dk := &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
								Args: tc.arguments,
							},
						},
					},
				},
			}
			if tc.expectedError == "" {
				assertAllowedWithoutWarnings(t, dk)
			} else {
				assertDenied(t, []string{tc.expectedError}, dk)
			}
		})
	}
}

func TestNoHostIdSourceArgument(t *testing.T) {
	type oneAgentArgumentTest struct {
		testName      string
		dk            dynakube.DynaKube
		expectedError string
	}

	testcases := []oneAgentArgumentTest{
		{
			testName: "host id source argument in cloud native full stack",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
								Args: []string{
									"--set-server=foo",
									"--set-host-id-source=foo",
								},
							},
						},
					},
				},
			},
			expectedError: errorHostIdSourceArgumentInCloudNative,
		},
		{
			testName: "no host id source argument in cloud native full stack",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
							HostInjectSpec: oneagent.HostInjectSpec{
								Args: []string{
									"--set-server=foo",
								},
							},
						},
					},
				},
			},
			expectedError: "",
		},
		{
			testName: "host id source argument in host monitoring stack",
			dk: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: testNamespace,
				},
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: oneagent.Spec{
						HostMonitoring: &oneagent.HostInjectSpec{
							Args: []string{
								"--set-server=foo",
								"--set-host-id-source=foo",
							},
						},
					},
				},
			},
			expectedError: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.expectedError == "" {
				assertAllowedWithoutWarnings(t, &tc.dk)
			} else {
				assertDenied(t, []string{tc.expectedError}, &tc.dk)
			}
		})
	}
}
