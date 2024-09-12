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

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNeedsCSIDriver(t *testing.T) {
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
						UseCSIDriver: useCSIDriver,
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

	t.Run(`cloud native fullstack with readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsCSIDriver())
	})

	t.Run(`cloud native fullstack without readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "false",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsCSIDriver())
	})

	t.Run(`host monitoring with readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsCSIDriver())
	})

	t.Run(`host monitoring without readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "false",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
		}
		assert.False(t, dk.NeedsCSIDriver())
	})
}

func TestNeedsReadonlyOneagent(t *testing.T) {
	t.Run(`cloud native fullstack default`, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsReadOnlyOneAgents())
	})

	t.Run(`host monitoring default`, func(t *testing.T) {
		dk := DynaKube{
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsReadOnlyOneAgents())
	})

	t.Run(`cloud native fullstack with readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsReadOnlyOneAgents())
	})

	t.Run(`cloud native fullstack without readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "false",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					CloudNativeFullStack: &CloudNativeFullStackSpec{},
				},
			},
		}
		assert.False(t, dk.NeedsReadOnlyOneAgents())
	})

	t.Run(`host monitoring with readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "true",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
		}
		assert.True(t, dk.NeedsReadOnlyOneAgents())
	})

	t.Run(`host monitoring without readonly host agent`, func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureReadOnlyOneAgent: "false",
				},
			},
			Spec: DynaKubeSpec{
				OneAgent: OneAgentSpec{
					HostMonitoring: &HostInjectSpec{},
				},
			},
		}
		assert.False(t, dk.NeedsReadOnlyOneAgents())
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
		dk := DynaKube{Spec: DynaKubeSpec{APIURL: testAPIURL}}

		assert.Equal(t, expectedImage, dk.DefaultOneAgentImage(version))
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
	dk := &DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-name",
		},
	}
	assert.Equal(t, "test-name-oneagent", dk.OneAgentDaemonsetName())
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

func TestIsOneAgentPrivileged(t *testing.T) {
	t.Run("is false by default", func(t *testing.T) {
		dk := DynaKube{}

		assert.False(t, dk.FeatureOneAgentPrivileged())
	})
	t.Run("is true when annotation is set to true", func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureRunOneAgentContainerPrivileged: "true",
				},
			},
		}

		assert.True(t, dk.FeatureOneAgentPrivileged())
	})
	t.Run("is false when annotation is set to false", func(t *testing.T) {
		dk := DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationFeatureRunOneAgentContainerPrivileged: "false",
				},
			},
		}

		assert.False(t, dk.FeatureOneAgentPrivileged())
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
