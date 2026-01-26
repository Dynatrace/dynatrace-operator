package daemonset

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUID   = "test-uid"
	testKey   = "test-key"
	testValue = "test-value"

	testClusterID    = "test-cluster-id"
	testURL          = "https://testing.dev.dynatracelabs.com/api"
	testDynakubeName = "test-dynakube-name"
	testName         = "test-name"

	testNewHostGroupName     = "newhostgroup"
	testOldHostGroupArgument = "--set-host-group=oldhostgroup"
	testNewHostGroupArgument = "--set-host-group=newhostgroup"
)

func TestArguments(t *testing.T) {
	t.Run("returns default arguments if hostInjection is nil", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{},
		}
		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("classic fullstack", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Args: []string{testValue},
					},
				},
			},
		}
		dsBuilder := classicFullStack{
			builder{
				dk:             &dk,
				hostInjectSpec: dk.Spec.OneAgent.ClassicFullStack,
				clusterID:      testClusterID,
			},
		}
		podSpecs, _ := dsBuilder.podSpec()
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)
		assert.Contains(t, podSpecs.Containers[0].Args, testValue)
	})
	t.Run("when injected arguments are provided then they are appended at the end of the arguments", func(t *testing.T) {
		args := []string{testValue}
		builder := builder{
			dk:             &dynakube.DynaKube{},
			hostInjectSpec: &oneagent.HostInjectSpec{Args: args},
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
			"test-value",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("when injected arguments are provided then they come last, only allowed duplicates and no duplicate key/value pairs", func(t *testing.T) {
		custArgs := []string{
			"--set-app-log-content-access=true",
			"--set-host-id-source=fqdn",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-server=https://hyper.super.com:9999",
			"--set-host-property=prop1",
			"--set-host-property=prop1",
			"--set-host-property=prop1",
			"--set-host-property=prop2",
			"--set-host-property=prop2",
			"--set-host-property=prop3",
			"--set-host-property=prop3",
			"--set-host-property=prop3",
			"--set-host-tag=tag1",
			"--set-host-tag=tag2",
			"--set-host-tag=tag2",
			"--set-host-tag=tag2",
			"--set-host-tag=tag3",
		}
		builder := builder{
			dk:             &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{}}}},
			hostInjectSpec: &oneagent.HostInjectSpec{Args: custArgs},
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-host-id-source=fqdn",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-host-property=prop1",
			"--set-host-property=prop2",
			"--set-host-property=prop3",
			"--set-host-tag=tag1",
			"--set-host-tag=tag2",
			"--set-host-tag=tag3",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server=https://hyper.super.com:9999",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("--set-proxy is not set with OneAgent version >=1.271.0", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					Proxy: &value.Source{Value: "something"},
				},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						VersionStatus: status.VersionStatus{
							Version: "1.285.0.20240122-141707",
						},
					},
				},
			},
		}
		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("proxy settings are not properly removed from OneAgent in case of feature-flag", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					Proxy: &value.Source{Value: "something"},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
					Annotations: map[string]string{
						"feature.dynatrace.com/oneagent-ignore-proxy": "true",
					},
				},
			},
		}
		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("proxy settings are not properly removed from OneAgent when we still have some left over", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
					Annotations: map[string]string{
						"feature.dynatrace.com/oneagent-ignore-proxy": "true",
					},
				},
				Spec: dynakube.DynaKubeSpec{Proxy: &value.Source{Value: testValue}},
			},
		}
		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("multiple set-host-property entries are possible", func(t *testing.T) {
		args := []string{
			"--set-app-log-content-access=true",
			"--set-host-id-source=lustiglustig",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-server=https://hyper.super.com:9999",
			"--set-host-property=item0=value0",
			"--set-host-property=item1=value1",
			"--set-host-property=item2=value2",
		}
		builder := builder{
			dk:             &dynakube.DynaKube{},
			hostInjectSpec: &oneagent.HostInjectSpec{Args: args},
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-host-id-source=lustiglustig",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-host-property=item0=value0",
			"--set-host-property=item1=value1",
			"--set-host-property=item2=value2",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server=https://hyper.super.com:9999",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("no-proxy is set", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					Proxy: &value.Source{Value: testValue},
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
					},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
					},
				},
			},
		}
		builder.dk.Annotations = map[string]string{
			exp.NoProxyKey: "*.dev.dynatracelabs.com",
		}

		arguments, _ := builder.arguments()

		assert.Contains(t, arguments, "--set-no-proxy=*.dev.dynatracelabs.com,dynakube-activegate.dynatrace")
	})
	t.Run("default no-proxy is set if AG is configured", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					Proxy: &value.Source{Value: testValue},
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
					},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
					},
				},
			},
		}

		arguments, _ := builder.arguments()

		assert.Contains(t, arguments, "--set-no-proxy=dynakube-activegate.dynatrace")
	})
	t.Run("allow arguments without value, but deduplicate", func(t *testing.T) {
		custArgs := []string{
			"--enable-feature-a",
			"--enable-feature-b",
			"--enable-feature-c",
			"--enable-feature-c",
			"--enable-feature-a",
			"--enable-feature-b",
		}
		builder := builder{
			dk:             &dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{}}}},
			hostInjectSpec: &oneagent.HostInjectSpec{Args: custArgs},
			deploymentType: deploymentmetadata.CloudNativeDeploymentType,
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--enable-feature-a",
			"--enable-feature-b",
			"--enable-feature-c",
			"--set-host-id-source=auto",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
}

func TestPodSpec_Arguments(t *testing.T) {
	dk := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{
					Args: []string{testKey, testValue, testUID},
				},
			},
		},
	}
	hostInjectSpecs := dk.Spec.OneAgent.ClassicFullStack
	dsBuilder := classicFullStack{
		builder{
			dk:             dk,
			hostInjectSpec: hostInjectSpecs,
			clusterID:      testClusterID,
			deploymentType: deploymentmetadata.ClassicFullStackDeploymentType,
		},
	}

	dk.Annotations = map[string]string{}
	podSpecs, _ := dsBuilder.podSpec()
	require.NotNil(t, podSpecs)
	require.NotEmpty(t, podSpecs.Containers)

	for _, arg := range hostInjectSpecs.Args {
		assert.Contains(t, podSpecs.Containers[0].Args, arg)
	}

	assert.Contains(t, podSpecs.Containers[0].Args, fmt.Sprintf("--set-host-property=OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))

	// deprecated
	t.Run("has proxy arg", func(t *testing.T) {
		dk.Status.OneAgent.Version = "1.272.0.0-0"
		dk.Spec.Proxy = &value.Source{Value: testValue}
		podSpecs, _ = dsBuilder.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")

		dk.Spec.Proxy = nil
		dk.Status.OneAgent.Version = ""
		podSpecs, _ = dsBuilder.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	// deprecated
	t.Run("has proxy arg but feature flag to ignore is enabled", func(t *testing.T) {
		dk.Spec.Proxy = &value.Source{Value: testValue}
		dk.Annotations[exp.OAProxyIgnoredKey] = "true"
		podSpecs, _ = dsBuilder.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	t.Run("has network zone arg", func(t *testing.T) {
		dk.Spec.NetworkZone = testValue
		podSpecs, _ = dsBuilder.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)

		dk.Spec.NetworkZone = ""
		podSpecs, _ = dsBuilder.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)
	})
	t.Run("has host-id-source arg for classic fullstack", func(t *testing.T) {
		daemonset, _ := dsBuilder.BuildDaemonSet()
		podSpecs = daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=auto")
	})
	t.Run("has host-id-source arg for hostMonitoring", func(t *testing.T) {
		hostMonInstance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{
						Args: []string{testKey, testValue, testUID},
					},
				},
			},
		}

		hostMonInjectSpec := hostMonInstance.Spec.OneAgent.HostMonitoring

		dsBuilder := hostMonitoring{
			builder{
				dk:             hostMonInstance,
				hostInjectSpec: hostMonInjectSpec,
				clusterID:      testClusterID,
			},
		}
		daemonset, _ := dsBuilder.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
	t.Run("has host-id-source arg for cloudNativeFullstack", func(t *testing.T) {
		cloudNativeInstance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{Args: []string{testKey, testValue, testUID}},
					},
				},
			},
		}

		dsBuilder := hostMonitoring{
			builder{
				dk:             cloudNativeInstance,
				hostInjectSpec: &cloudNativeInstance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
				clusterID:      testClusterID,
			},
		}
		daemonset, _ := dsBuilder.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
	t.Run("has host-group for classicFullstack", func(t *testing.T) {
		classicInstance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostGroup: testNewHostGroupName,
					ClassicFullStack: &oneagent.HostInjectSpec{
						Args: []string{testOldHostGroupArgument},
					},
				},
			},
		}

		dsBuilder := hostMonitoring{
			builder{
				dk:             classicInstance,
				hostInjectSpec: classicInstance.Spec.OneAgent.ClassicFullStack,
			},
		}
		arguments, err := dsBuilder.arguments()
		require.NoError(t, err)
		assert.Contains(t, arguments, testNewHostGroupArgument)
	})
	t.Run("has host-group for cloudNativeFullstack", func(t *testing.T) {
		cloudNativeInstance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostGroup: testNewHostGroupName,
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{Args: []string{testOldHostGroupArgument}},
					},
				},
			},
		}

		dsBuilder := hostMonitoring{
			builder{
				dk:             cloudNativeInstance,
				hostInjectSpec: &cloudNativeInstance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			},
		}
		arguments, err := dsBuilder.arguments()
		require.NoError(t, err)
		assert.Contains(t, arguments, testNewHostGroupArgument)
	})
	t.Run("has host-group for HostMonitoring", func(t *testing.T) {
		hostMonitoringInstance := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostGroup: testNewHostGroupName,
					HostMonitoring: &oneagent.HostInjectSpec{
						Args: []string{testOldHostGroupArgument},
					},
				},
			},
		}

		dsBuilder := hostMonitoring{
			builder{
				dk:             hostMonitoringInstance,
				hostInjectSpec: hostMonitoringInstance.Spec.OneAgent.HostMonitoring,
			},
		}
		arguments, err := dsBuilder.arguments()
		require.NoError(t, err)
		assert.Contains(t, arguments, testNewHostGroupArgument)
	})
}
