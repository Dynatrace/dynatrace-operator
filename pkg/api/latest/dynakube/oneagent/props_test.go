/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package oneagent

import (
	"net/url"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const testAPIURL = "http://test-endpoint/api"

func TestNeedsReadonlyOneagent(t *testing.T) {
	t.Run("cloud native fullstack always use readonly host agent", func(t *testing.T) {
		oneagent := OneAgent{
			Spec: &Spec{
				CloudNativeFullStack: &CloudNativeFullStackSpec{},
			},
		}
		assert.True(t, oneagent.IsReadOnlyFSSupported())
	})

	t.Run("host monitoring with readonly host agent", func(t *testing.T) {
		oneAgent := OneAgent{
			Spec: &Spec{
				HostMonitoring: &HostInjectSpec{},
			},
		}
		assert.True(t, oneAgent.IsReadOnlyFSSupported())
	})

	t.Run("host monitoring without readonly host agent", func(t *testing.T) {
		setupDisabledCSIEnv(t)

		oneAgent := OneAgent{
			Spec: &Spec{
				HostMonitoring: &HostInjectSpec{},
			},
		}
		assert.True(t, oneAgent.IsReadOnlyFSSupported())
	})
}

func TestDefaultOneAgentImage(t *testing.T) {
	t.Run("OneAgentImage with no API URL", func(t *testing.T) {
		oneAgent := OneAgent{}
		assert.Empty(t, oneAgent.GetDefaultImage(""))
	})

	t.Run("OneAgentImage adds raw postfix", func(t *testing.T) {
		hostURL, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostURL.Host, false, false, false)
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", oneAgent.GetDefaultImage("1.234.5"))
	})

	t.Run("OneAgentImage doesn't add 'raw' postfix if present", func(t *testing.T) {
		hostURL, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostURL.Host, false, false, false)
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", oneAgent.GetDefaultImage("1.234.5-raw"))
	})

	t.Run("OneAgentImage with custom version truncates build date", func(t *testing.T) {
		version := "1.239.14.20220325-164521"
		expectedImage := "test-endpoint/linux/oneagent:1.239.14-raw"
		hostURL, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostURL.Host, false, false, false)
		assert.Equal(t, expectedImage, oneAgent.GetDefaultImage(version))
	})
}

func TestCustomOneAgentImage(t *testing.T) {
	t.Run("OneAgentImage with custom image", func(t *testing.T) {
		customImg := "registry/my/oneagent:latest"
		oneAgent := OneAgent{Spec: &Spec{ClassicFullStack: &HostInjectSpec{Image: customImg}}}
		assert.Equal(t, customImg, oneAgent.GetCustomImage())
	})

	t.Run("OneAgentImage with no custom image", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{ClassicFullStack: &HostInjectSpec{}}}
		assert.Empty(t, oneAgent.GetCustomImage())
	})
}

func TestOneAgentDaemonsetName(t *testing.T) {
	oneAgent := OneAgent{name: "test-name"}
	assert.Equal(t, "test-name-oneagent", oneAgent.GetDaemonsetName())
}

func TestCodeModulesVersion(t *testing.T) {
	testVersion := "1.2.3"

	t.Run("use status", func(t *testing.T) {
		codeModulesStatus := &CodeModulesStatus{VersionStatus: status.VersionStatus{Version: testVersion}}
		oneAgent := NewOneAgent(&Spec{}, &Status{}, codeModulesStatus, "", "", false, false, false)
		version := oneAgent.GetCodeModulesVersion()
		assert.Equal(t, testVersion, version)
	})
	t.Run("use version ", func(t *testing.T) {
		codeModulesStatus := &CodeModulesStatus{VersionStatus: status.VersionStatus{Version: "other"}}
		oneAgent := NewOneAgent(&Spec{
			ApplicationMonitoring: &ApplicationMonitoringSpec{Version: testVersion},
		}, &Status{}, codeModulesStatus, "", "", false, false, false)
		version := oneAgent.GetCustomCodeModulesVersion()

		assert.Equal(t, testVersion, version)
	})
}

func TestGetOneAgentEnvironment(t *testing.T) {
	t.Run("get environment from classicFullstack", func(t *testing.T) {
		oneAgent := OneAgent{
			Spec: &Spec{
				ClassicFullStack: &HostInjectSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "classicFullstack",
							Value: "true",
						},
					},
				},
			},
		}
		env := oneAgent.GetEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "classicFullstack", env[0].Name)
	})

	t.Run("get environment from hostMonitoring", func(t *testing.T) {
		oneAgent := OneAgent{
			Spec: &Spec{
				HostMonitoring: &HostInjectSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "hostMonitoring",
							Value: "true",
						},
					},
				},
			},
		}
		env := oneAgent.GetEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "hostMonitoring", env[0].Name)
	})

	t.Run("get environment from cloudNative", func(t *testing.T) {
		oneAgent := OneAgent{
			Spec: &Spec{
				CloudNativeFullStack: &CloudNativeFullStackSpec{
					HostInjectSpec: HostInjectSpec{
						Env: []corev1.EnvVar{
							{
								Name:  "cloudNative",
								Value: "true",
							},
						},
					},
				},
			},
		}
		env := oneAgent.GetEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "cloudNative", env[0].Name)
	})

	t.Run("get environment from applicationMonitoring", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{
			ApplicationMonitoring: &ApplicationMonitoringSpec{},
		}}
		env := oneAgent.GetEnvironment()

		require.NotNil(t, env)
		assert.Empty(t, env)
	})

	t.Run("get environment from unconfigured dynakube", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{}}
		env := oneAgent.GetEnvironment()

		require.NotNil(t, env)
		assert.Empty(t, env)
	})
}

func TestOneAgentHostGroup(t *testing.T) {
	t.Run("get host group from cloudNativeFullstack.args", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--set-host-group=arg",
					},
				},
			},
		},
		}
		hostGroup := dk.GetHostGroup()
		assert.Equal(t, "arg", hostGroup)
	})

	t.Run("get host group from oneagent.hostGroup", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			HostGroup: "field",
		},
		}
		hostGroup := dk.GetHostGroup()
		assert.Equal(t, "field", hostGroup)
	})

	t.Run("get host group if both methods used", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--set-host-group=arg",
					},
				},
			},
			HostGroup: "field",
		},
		}
		hostGroup := dk.GetHostGroup()
		assert.Equal(t, "field", hostGroup)
	})
}

func TestOneAgentArgumentsMap(t *testing.T) {
	t.Run("straight forward argument list", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--set-host-id-source=k8s-node-name",
						"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
						"--set-host-property=dt.security_context=kubernetes_clusters",
						"--set-host-property=dynakube-name=$(CUSTOM_CRD_NAME)",
						"--set-no-proxy=",
						"--set-proxy=",
						"--set-tenant=$(DT_TENANT)",
						"--set-server=dynatrace.com",
						"--set-host-property=prop1=val1",
						"--set-host-property=prop2=val2",
						"--set-host-property=prop3=val3",
						"--set-host-tag=tag1",
						"--set-host-tag=tag2",
						"--set-host-tag=tag3",
					},
				},
			},
			HostGroup: "field",
		},
		}
		argMap := dk.GetArgumentsMap()
		require.Len(t, argMap, 7)

		require.Len(t, argMap["--set-host-id-source"], 1)
		assert.Equal(t, "k8s-node-name", argMap["--set-host-id-source"][0])

		require.Len(t, argMap["--set-host-property"], 6)
		assert.Equal(t, "OperatorVersion=$(DT_OPERATOR_VERSION)", argMap["--set-host-property"][0])
		assert.Equal(t, "dt.security_context=kubernetes_clusters", argMap["--set-host-property"][1])
		assert.Equal(t, "dynakube-name=$(CUSTOM_CRD_NAME)", argMap["--set-host-property"][2])
		assert.Equal(t, "prop1=val1", argMap["--set-host-property"][3])
		assert.Equal(t, "prop2=val2", argMap["--set-host-property"][4])
		assert.Equal(t, "prop3=val3", argMap["--set-host-property"][5])

		require.Len(t, argMap["--set-no-proxy"], 1)
		assert.Empty(t, argMap["--set-no-proxy"][0])

		require.Len(t, argMap["--set-proxy"], 1)
		assert.Empty(t, argMap["--set-proxy"][0])

		require.Len(t, argMap["--set-tenant"], 1)
		assert.Equal(t, "$(DT_TENANT)", argMap["--set-tenant"][0])

		require.Len(t, argMap["--set-server"], 1)
		assert.Equal(t, "dynatrace.com", argMap["--set-server"][0])
	})

	t.Run("multiple --set-host-property arguments", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--set-host-property=prop1=val1",
						"--set-host-property=prop2=val2",
						"--set-host-property=prop3=val3",
						"--set-host-property=prop3=val3",
					},
				},
			},
			HostGroup: "field",
		},
		}
		argMap := dk.GetArgumentsMap()
		require.Len(t, argMap, 1)
		require.Len(t, argMap["--set-host-property"], 4)

		assert.Equal(t, "prop1=val1", argMap["--set-host-property"][0])
		assert.Equal(t, "prop2=val2", argMap["--set-host-property"][1])
		assert.Equal(t, "prop3=val3", argMap["--set-host-property"][2])
	})

	t.Run("multiple --set-host-tag arguments", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--set-host-tag=tag1=1",
						"--set-host-tag=tag1=2",
						"--set-host-tag=tag1=3",
						"--set-host-tag=tag2",
						"--set-host-tag=tag3",
					},
				},
			},
			HostGroup: "field",
		},
		}
		argMap := dk.GetArgumentsMap()
		require.Len(t, argMap, 1)
		require.Len(t, argMap["--set-host-tag"], 5)

		assert.Equal(t, "tag1=1", argMap["--set-host-tag"][0])
		assert.Equal(t, "tag1=2", argMap["--set-host-tag"][1])
		assert.Equal(t, "tag1=3", argMap["--set-host-tag"][2])
		assert.Equal(t, "tag2", argMap["--set-host-tag"][3])
		assert.Equal(t, "tag3", argMap["--set-host-tag"][4])
	})

	t.Run("arguments without value", func(t *testing.T) {
		dk := OneAgent{Spec: &Spec{
			CloudNativeFullStack: &CloudNativeFullStackSpec{
				HostInjectSpec: HostInjectSpec{
					Args: []string{
						"--enable-feature-a",
						"--enable-feature-b",
						"--enable-feature-c",
					},
				},
			},
			HostGroup: "field",
		},
		}
		argMap := dk.GetArgumentsMap()
		require.Len(t, argMap, 3)
		require.Len(t, argMap["--enable-feature-a"], 1)
		require.Len(t, argMap["--enable-feature-b"], 1)
		require.Len(t, argMap["--enable-feature-c"], 1)
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

func TestOneAgent_IsAutoUpdateEnabled(t *testing.T) {
	type testcase struct {
		name              string
		spec              *Spec
		autoUpdateEnabled bool
	}

	t.Run("classic, cloud, hostmonitoring", func(t *testing.T) {
		tcs := []testcase{
			{
				name: "Classic Full Stack",
				spec: &Spec{
					ClassicFullStack: &HostInjectSpec{
						Version: "",
						Image:   "",
					},
				},
				autoUpdateEnabled: true,
			},
			{
				name: "Classic Full Stack - version",
				spec: &Spec{
					ClassicFullStack: &HostInjectSpec{
						Version: "version",
						Image:   "",
					},
				},
				autoUpdateEnabled: false,
			},
			{
				name: "Classic Full Stack - image",
				spec: &Spec{
					ClassicFullStack: &HostInjectSpec{
						Version: "",
						Image:   "image",
					},
				},
				autoUpdateEnabled: false,
			},
			{
				name: "Cloud Native Full Stack",
				spec: &Spec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
				autoUpdateEnabled: true,
			},
			{
				name: "Cloud Native Full Stack - version",
				spec: &Spec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{
						HostInjectSpec: HostInjectSpec{
							Version: "version",
							Image:   "",
						},
					},
				},
				autoUpdateEnabled: false,
			},
			{
				name: "Cloud Native Full Stack - image",
				spec: &Spec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{
						HostInjectSpec: HostInjectSpec{
							Version: "",
							Image:   "image",
						},
					},
				},
				autoUpdateEnabled: false,
			},
			{
				name: "Host Monitoring",
				spec: &Spec{
					HostMonitoring: &HostInjectSpec{},
				},
				autoUpdateEnabled: true,
			},
			{
				name: "Host Monitoring",
				spec: &Spec{
					HostMonitoring: &HostInjectSpec{
						Version: "version",
						Image:   "",
					},
				},
				autoUpdateEnabled: false,
			},
			{
				name: "Host Monitoring",
				spec: &Spec{
					HostMonitoring: &HostInjectSpec{
						Version: "",
						Image:   "image",
					},
				},
				autoUpdateEnabled: false,
			},
		}

		for _, tc := range tcs {
			oa := NewOneAgent(tc.spec, nil, nil, "", "", false, false, false)
			assert.Equal(t, tc.autoUpdateEnabled, oa.IsAutoUpdateEnabled(), tc.name)
		}
	})

	t.Run("application monitoring", func(t *testing.T) {
		tc := testcase{
			name: "Application Monitoring",
			spec: &Spec{
				ApplicationMonitoring: &ApplicationMonitoringSpec{},
			},
			autoUpdateEnabled: false,
		}

		oa := NewOneAgent(tc.spec, nil, nil, "", "", false, false, false)
		assert.Equal(t, tc.autoUpdateEnabled, oa.IsAutoUpdateEnabled(), tc.name)
	})
}

func TestOneAgent_GetHostPath(t *testing.T) {
	tenant := "tenant"

	tcs := []struct {
		name             string
		oa               *OneAgent
		expectedHostPath string
	}{
		{
			name: "Classic Full Stack - ignored",
			oa: &OneAgent{
				Spec: &Spec{
					ClassicFullStack: &HostInjectSpec{
						StorageHostPath: "whatever",
					},
				},
			},
			expectedHostPath: "",
		},
		{
			name: "Cloud Native Full Stack - custom host path, append tenant",
			oa: &OneAgent{
				Spec: &Spec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{
						HostInjectSpec: HostInjectSpec{
							StorageHostPath: "/something/custom",
						},
					},
				},
			},
			expectedHostPath: filepath.Join("/something/custom", tenant),
		},
		{
			name: "Host Monitoring - custom host path, append tenant",
			oa: &OneAgent{
				Spec: &Spec{
					HostMonitoring: &HostInjectSpec{
						StorageHostPath: "/something/custom",
					},
				},
			},
			expectedHostPath: filepath.Join("/something/custom", tenant),
		},
		{
			name: "Host Monitoring - default host path",
			oa: &OneAgent{
				Spec: &Spec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
			expectedHostPath: filepath.Join(StorageVolumeDefaultHostPath, tenant),
		},
		{
			name: "Cloud Native Full Stack - default host path",
			oa: &OneAgent{
				Spec: &Spec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
			expectedHostPath: filepath.Join(StorageVolumeDefaultHostPath, tenant),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			oa := tc.oa
			assert.Equal(t, tc.expectedHostPath, oa.GetHostPath(tenant))
		})
	}
}
