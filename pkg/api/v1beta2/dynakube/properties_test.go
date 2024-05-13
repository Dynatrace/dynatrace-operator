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

package dynakube

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testAPIURL = "http://test-endpoint/api"

func TestDefaultActiveGateImage(t *testing.T) {
	t.Run(`ActiveGateImage with no API URL`, func(t *testing.T) {
		dk := DynaKube{}
		assert.Equal(t, "", dk.DefaultActiveGateImage(""))
	})

	t.Run(`ActiveGateImage adds raw postfix`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/activegate:1.234.5-raw", dk.DefaultActiveGateImage("1.234.5"))
	})

	t.Run("ActiveGateImage doesn't add 'raw' postfix if present", func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/activegate:1.234.5-raw", dk.DefaultActiveGateImage("1.234.5-raw"))
	})

	t.Run(`ActiveGateImage truncates build date`, func(t *testing.T) {
		version := "1.239.14.20220325-164521"
		expectedImage := "test-endpoint/linux/activegate:1.239.14-raw"
		dynakube := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}

		assert.Equal(t, expectedImage, dynakube.DefaultActiveGateImage(version))
	})
}

func TestCustomActiveGateImage(t *testing.T) {
	t.Run(`ActiveGateImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/activegate:latest"
		dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{CapabilityProperties: CapabilityProperties{
			Image: customImg,
		}}}}
		assert.Equal(t, customImg, dk.CustomActiveGateImage())
	})

	t.Run(`ActiveGateImage with no custom image`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{ActiveGate: ActiveGateSpec{CapabilityProperties: CapabilityProperties{}}}}
		assert.Equal(t, "", dk.CustomActiveGateImage())
	})
}

func TestDynaKube_UseCSIDriver(t *testing.T) {
	t.Run(`DynaKube with application monitoring without csi driver`, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{},
				},
			},
		}
		assert.False(t, dk.NeedsCSIDriver())
	})

	t.Run(`DynaKube with application monitoring with csi driver enabled`, func(t *testing.T) {
		useCSIDriver := true
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{
						UseCSIDriver: &useCSIDriver,
					},
				},
			},
		}
		assert.True(t, dk.NeedsCSIDriver())
	})

	t.Run(`DynaKube with cloud native`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{CloudNativeFullStack: &CloudNativeFullStackSpec{}}}}
		assert.True(t, dk.NeedsCSIDriver())
	})
}

func TestDefaultOneAgentImage(t *testing.T) {
	t.Run(`OneAgentImage with no API URL`, func(t *testing.T) {
		dk := DynaKube{}
		assert.Equal(t, "", dk.DefaultOneAgentImage(""))
	})

	t.Run(`OneAgentImage adds raw postfix`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", dk.DefaultOneAgentImage("1.234.5"))
	})

	t.Run(`OneAgentImage doesn't add 'raw' postfix if present`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}
		assert.Equal(t, "test-endpoint/linux/oneagent:1.234.5-raw", dk.DefaultOneAgentImage("1.234.5-raw"))
	})

	t.Run(`OneAgentImage with custom version truncates build date`, func(t *testing.T) {
		version := "1.239.14.20220325-164521"
		expectedImage := "test-endpoint/linux/oneagent:1.239.14-raw"
		dynakube := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}

		assert.Equal(t, expectedImage, dynakube.DefaultOneAgentImage(version))
	})
}

func TestCustomOneAgentImage(t *testing.T) {
	t.Run(`OneAgentImage with custom image`, func(t *testing.T) {
		customImg := "registry/my/oneagent:latest"
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{ClassicFullStack: &HostInjectSpec{Image: customImg}}}}
		assert.Equal(t, customImg, dk.CustomOneAgentImage())
	})

	t.Run(`OneAgentImage with no custom image`, func(t *testing.T) {
		dk := DynaKube{Spec: DynaKubeSpec{OneAgent: OneAgentSpec{ClassicFullStack: &HostInjectSpec{}}}}
		assert.Equal(t, "", dk.CustomOneAgentImage())
	})
}

func TestOneAgentDaemonsetName(t *testing.T) {
	instance := &DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
	}
	assert.Equal(t, "test-name-oneagent", instance.OneAgentDaemonsetName())
}

func TestTokens(t *testing.T) {
	testName := "test-name"
	testValue := "test-value"

	t.Run(`GetTokensName returns custom token name`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName},
			Spec:       DynaKubeSpec{Tokens: testValue},
		}
		assert.Equal(t, dk.Tokens(), testValue)
	})
	t.Run(`GetTokensName uses instance name as default value`, func(t *testing.T) {
		dk := DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testName}}
		assert.Equal(t, dk.Tokens(), testName)
	})
}

func TestTenantUUID(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		apiUrl := "https://demo.dev.dynatracelabs.com/api"
		expectedTenantId := "demo"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("happy path (alternative, no domain)", func(t *testing.T) {
		apiUrl := "https://dynakube-activegate/e/tenant/api/v2/metrics/ingest"
		expectedTenantId := "tenant"

		actualTenantId, err := tenantUUID(apiUrl)

		require.NoErrorf(t, err, "Expected that getting tenant id from '%s' will be successful", apiUrl)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("missing API URL protocol", func(t *testing.T) {
		apiUrl := "demo.dev.dynatracelabs.com/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from API URL 'demo.dev.dynatracelabs.com/api'"

		actualTenantId, err := tenantUUID(apiUrl)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})

	t.Run("suffix-only, relative API URL", func(t *testing.T) {
		apiUrl := "/api"
		expectedTenantId := ""
		expectedError := "problem getting tenant id from API URL '/api'"

		actualTenantId, err := tenantUUID(apiUrl)

		require.EqualErrorf(t, err, expectedError, "Expected that getting tenant id from '%s' will result in: '%v'",
			apiUrl, expectedError,
		)
		assert.Equalf(t, expectedTenantId, actualTenantId, "Expected that tenant id of %s is %s, but found %s",
			apiUrl, expectedTenantId, actualTenantId,
		)
	})
}

func TestCodeModulesVersion(t *testing.T) {
	testVersion := "1.2.3"

	t.Run(`use status`, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
			Status: DynaKubeStatus{
				CodeModules: CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: testVersion,
					},
				},
			},
		}
		version := dk.CodeModulesVersion()
		assert.Equal(t, testVersion, version)
	})
	t.Run(`use version `, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{
						Version: testVersion,
					},
				},
			},
			Status: DynaKubeStatus{
				CodeModules: CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "other",
					},
				},
			},
		}
		version := dk.CustomCodeModulesVersion()
		assert.Equal(t, testVersion, version)
	})
}

func TestGetRawImageTag(t *testing.T) {
	t.Run(`with tag`, func(t *testing.T) {
		expectedTag := "test"
		rawTag := getRawImageTag("example.test:" + expectedTag)
		require.Equal(t, expectedTag, rawTag)
	})
	t.Run(`without tag`, func(t *testing.T) {
		expectedTag := "latest"
		rawTag := getRawImageTag("example.test")
		require.Equal(t, expectedTag, rawTag)
	})
	t.Run(`local URI with port`, func(t *testing.T) {
		expectedTag := "test"
		// based on https://docs.docker.com/engine/reference/commandline/tag/#tag-an-image-for-a-private-repository
		rawTag := getRawImageTag("myregistryhost:5000/fedora/httpd:" + expectedTag)
		require.Equal(t, expectedTag, rawTag)
	})
	t.Run(`wrong URI => no panic`, func(t *testing.T) {
		rawTag := getRawImageTag("example.test:")
		require.Equal(t, "", rawTag)
	})
	t.Run(`very wrong URI => no panic`, func(t *testing.T) {
		rawTag := getRawImageTag(":")
		require.Equal(t, "", rawTag)
	})
}

func TestIsOneAgentPrivileged(t *testing.T) {
	t.Run("is false by default", func(t *testing.T) {
		dynakube := DynaKube{}

		assert.False(t, dynakube.FeatureOneAgentPrivileged())
	})
	t.Run("is true when annotation is set to true", func(t *testing.T) {
		dynakube := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureRunOneAgentContainerPrivileged: "true",
				},
			},
		}

		assert.True(t, dynakube.FeatureOneAgentPrivileged())
	})
	t.Run("is false when annotation is set to false", func(t *testing.T) {
		dynakube := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureRunOneAgentContainerPrivileged: "false",
				},
			},
		}

		assert.False(t, dynakube.FeatureOneAgentPrivileged())
	})
}

func TestGetOneAgentEnvironment(t *testing.T) {
	t.Run("get environment from classicFullstack", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ClassicFullStack: &HostInjectSpec{
						Env: []corev1.EnvVar{
							{
								Name:  "classicFullstack",
								Value: "true",
							},
						},
					},
				},
			},
		}
		env := dk.GetOneAgentEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "classicFullstack", env[0].Name)
	})

	t.Run("get environment from hostMonitoring", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{
						Env: []corev1.EnvVar{
							{
								Name:  "hostMonitoring",
								Value: "true",
							},
						},
					},
				},
			},
		}
		env := dk.GetOneAgentEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "hostMonitoring", env[0].Name)
	})

	t.Run("get environment from cloudNative", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
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
			},
		}
		env := dk.GetOneAgentEnvironment()

		require.Len(t, env, 1)
		assert.Equal(t, "cloudNative", env[0].Name)
	})

	t.Run("get environment from applicationMonitoring", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					ApplicationMonitoring: &ApplicationMonitoringSpec{},
				},
			},
		}
		env := dk.GetOneAgentEnvironment()

		require.NotNil(t, env)
		assert.Empty(t, env)
	})

	t.Run("get environment from unconfigured dynakube", func(t *testing.T) {
		dk := DynaKube{}
		env := dk.GetOneAgentEnvironment()

		require.NotNil(t, env)
		assert.Empty(t, env)
	})
}

func TestDynaKube_ShallUpdateActiveGateConnectionInfo(t *testing.T) {
	dk := DynaKube{
		Status: DynaKubeStatus{
			DynatraceApi: DynatraceApiStatus{
				LastTokenScopeRequest: metav1.Time{},
			},
			OneAgent: OneAgentStatus{
				ConnectionInfoStatus: OneAgentConnectionInfoStatus{
					ConnectionInfoStatus: ConnectionInfoStatus{},
				},
			},
			ActiveGate: ActiveGateStatus{
				ConnectionInfoStatus: ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: ConnectionInfoStatus{},
				},
			},
		},
	}

	timeProvider := timeprovider.New().Freeze()
	tests := map[string]struct {
		lastRequestTimeDeltaMinutes int
		updateExpected              bool
		threshold                   int
	}{
		"Do not update after 10 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -10,
			updateExpected:              false,
			threshold:                   -1,
		},
		"Do update after 20 minutes using default interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   -1,
		},
		"Do not update after 3 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -3,
			updateExpected:              false,
			threshold:                   5,
		},
		"Do update after 7 minutes using 5m interval": {
			lastRequestTimeDeltaMinutes: -7,
			updateExpected:              true,
			threshold:                   5,
		},
		"Do not update after 17 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -17,
			updateExpected:              false,
			threshold:                   20,
		},
		"Do update after 22 minutes using 20m interval": {
			lastRequestTimeDeltaMinutes: -22,
			updateExpected:              true,
			threshold:                   20,
		},
		"Do update immediately using 0m interval": {
			lastRequestTimeDeltaMinutes: 0,
			updateExpected:              true,
			threshold:                   0,
		},
		"Do update after 1 minute using 0m interval": {
			lastRequestTimeDeltaMinutes: -1,
			updateExpected:              true,
			threshold:                   0,
		},
		"Do update after 20 minutes using 0m interval": {
			lastRequestTimeDeltaMinutes: -20,
			updateExpected:              true,
			threshold:                   0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk.Spec.DynatraceApiRequestThreshold = address.Of(time.Duration(test.threshold))

			lastRequestTime := timeProvider.Now().Add(time.Duration(test.lastRequestTimeDeltaMinutes) * time.Minute)
			dk.Status.DynatraceApi.LastTokenScopeRequest.Time = lastRequestTime

			assert.Equal(t, test.updateExpected, dk.IsTokenScopeVerificationAllowed(timeProvider))
		})
	}
}

func TestOneAgentHostGroup(t *testing.T) {
	t.Run("get host group from cloudNativeFullstack.args", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{
						HostInjectSpec: HostInjectSpec{
							Args: []string{
								"--set-host-group=arg",
							},
						},
					},
				},
			},
		}
		hostGroup := dk.HostGroup()
		assert.Equal(t, "arg", hostGroup)
	})

	t.Run("get host group from oneagent.hostGroup", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostGroup: "field",
				},
			},
		}
		hostGroup := dk.HostGroup()
		assert.Equal(t, "field", hostGroup)
	})

	t.Run("get host group if both methods used", func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{
						HostInjectSpec: HostInjectSpec{
							Args: []string{
								"--set-host-group=arg",
							},
						},
					},
					HostGroup: "field",
				},
			},
		}
		hostGroup := dk.HostGroup()
		assert.Equal(t, "field", hostGroup)
	})
}
