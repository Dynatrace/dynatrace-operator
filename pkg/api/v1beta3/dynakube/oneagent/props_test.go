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
		assert.True(t, oneagent.UseReadOnlyOneAgents())
	})

	t.Run("host monitoring with readonly host agent", func(t *testing.T) {
		oneAgent := OneAgent{
			Spec: &Spec{
				HostMonitoring: &HostInjectSpec{},
			},
		}
		assert.True(t, oneAgent.UseReadOnlyOneAgents())
	})

	t.Run("host monitoring without readonly host agent", func(t *testing.T) {
		setupDisabledCSIEnv(t)

		oneAgent := OneAgent{
			Spec: &Spec{
				HostMonitoring: &HostInjectSpec{},
			},
		}
		assert.False(t, oneAgent.UseReadOnlyOneAgents())
	})
}

func TestDefaultOneAgentImage(t *testing.T) {
	t.Run("OneAgentImage with no API URL", func(t *testing.T) {
		oneAgent := OneAgent{}
		assert.Equal(t, "", oneAgent.DefaultOneAgentImage(""))
	})

	t.Run("OneAgentImage adds raw postfix", func(t *testing.T) {
		hostUrl, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostUrl.Host, false, false)
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", oneAgent.DefaultOneAgentImage("1.234.5"))
	})

	t.Run("OneAgentImage doesn't add 'raw' postfix if present", func(t *testing.T) {
		hostUrl, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostUrl.Host, false, false)
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", oneAgent.DefaultOneAgentImage("1.234.5-raw"))
	})

	t.Run("OneAgentImage with custom version truncates build date", func(t *testing.T) {
		version := "1.239.14.20220325-164521"
		expectedImage := "test-endpoint/linux/oneagent:1.239.14-raw"
		hostUrl, _ := url.Parse(testAPIURL)
		oneAgent := NewOneAgent(&Spec{}, &Status{}, &CodeModulesStatus{}, "", hostUrl.Host, false, false)
		assert.Equal(t, expectedImage, oneAgent.DefaultOneAgentImage(version))
	})
}

func TestCustomOneAgentImage(t *testing.T) {
	t.Run("OneAgentImage with custom image", func(t *testing.T) {
		customImg := "registry/my/oneagent:latest"
		oneAgent := OneAgent{Spec: &Spec{ClassicFullStack: &HostInjectSpec{Image: customImg}}}
		assert.Equal(t, customImg, oneAgent.CustomOneAgentImage())
	})

	t.Run("OneAgentImage with no custom image", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{ClassicFullStack: &HostInjectSpec{}}}
		assert.Equal(t, "", oneAgent.CustomOneAgentImage())
	})
}

func TestOneAgentDaemonsetName(t *testing.T) {
	oneAgent := OneAgent{name: "test-name"}
	assert.Equal(t, "test-name-oneagent", oneAgent.OneAgentDaemonsetName())
}

func TestCodeModulesVersion(t *testing.T) {
	testVersion := "1.2.3"

	t.Run("use status", func(t *testing.T) {
		codeModulesStatus := &CodeModulesStatus{VersionStatus: status.VersionStatus{Version: testVersion}}
		oneAgent := NewOneAgent(&Spec{}, &Status{}, codeModulesStatus, "", "", false, false)
		version := oneAgent.CodeModulesVersion()
		assert.Equal(t, testVersion, version)
	})
	t.Run("use version ", func(t *testing.T) {
		codeModulesStatus := &CodeModulesStatus{VersionStatus: status.VersionStatus{Version: "other"}}
		oneAgent := NewOneAgent(&Spec{
			ApplicationMonitoring: &ApplicationMonitoringSpec{Version: testVersion},
		}, &Status{}, codeModulesStatus, "", "", false, false)
		version := oneAgent.CustomCodeModulesVersion()

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
		env := oneAgent.GetOneAgentEnvironment()

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
		env := oneAgent.GetOneAgentEnvironment()

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
		env := oneAgent.GetOneAgentEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "cloudNative", env[0].Name)
	})

	t.Run("get environment from applicationMonitoring", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{
			ApplicationMonitoring: &ApplicationMonitoringSpec{},
		}}
		env := oneAgent.GetOneAgentEnvironment()

		require.NotNil(t, env)
		assert.Empty(t, env)
	})

	t.Run("get environment from unconfigured dynakube", func(t *testing.T) {
		oneAgent := OneAgent{Spec: &Spec{}}
		env := oneAgent.GetOneAgentEnvironment()

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
