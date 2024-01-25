package daemonset

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUID   = "test-uid"
	testKey   = "test-key"
	testValue = "test-value"

	testClusterID = "test-cluster-id"
	testURL       = "https://testing.dev.dynatracelabs.com/api"
	testName      = "test-name"
)

func TestArguments(t *testing.T) {
	t.Run("returns default arguments if hostInjection is nil", func(t *testing.T) {
		builder := builderInfo{
			dynakube: &dynatracev1beta1.DynaKube{},
		}
		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("classic fullstack", func(t *testing.T) {
		instance := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						Args: []string{testValue},
					},
				},
			},
		}
		dsInfo := ClassicFullStack{
			builderInfo{
				dynakube:       &instance,
				hostInjectSpec: instance.Spec.OneAgent.ClassicFullStack,
				clusterID:      testClusterID,
			},
		}
		podSpecs, _ := dsInfo.podSpec()
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)
		assert.Contains(t, podSpecs.Containers[0].Args, testValue)
	})
	t.Run("when injected arguments are provided then they are appended at the end of the arguments", func(t *testing.T) {
		args := []string{testValue}
		builder := builderInfo{
			dynakube:       &dynatracev1beta1.DynaKube{},
			hostInjectSpec: &dynatracev1beta1.HostInjectSpec{Args: args},
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-proxy=",
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
			"test-value",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("when injected arguments are provided then they take precedence", func(t *testing.T) {
		args := []string{
			"--set-app-log-content-access=true",
			"--set-host-id-source=lustiglustig",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-server=https://hyper.super.com:9999",
		}
		builder := builderInfo{
			dynakube:       &dynatracev1beta1.DynaKube{},
			hostInjectSpec: &dynatracev1beta1.HostInjectSpec{Args: args},
		}

		arguments, _ := builder.arguments()

		expectedDefaultArguments := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-host-id-source=lustiglustig",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-proxy=",
			"--set-server=https://hyper.super.com:9999",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
	t.Run("--set-proxy is not set with OneAgent version >=1.271.0", func(t *testing.T) {
		builder := builderInfo{
			dynakube: &dynatracev1beta1.DynaKube{
				Status: dynatracev1beta1.DynaKubeStatus{
					OneAgent: dynatracev1beta1.OneAgentStatus{
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
			"--set-server={$(DT_SERVER)}",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, arguments)
	})
}

func TestPodSpec_Arguments(t *testing.T) {
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
					Args: []string{testKey, testValue, testUID},
				},
			},
		},
	}
	hostInjectSpecs := instance.Spec.OneAgent.ClassicFullStack
	dsInfo := ClassicFullStack{
		builderInfo{
			dynakube:       instance,
			hostInjectSpec: hostInjectSpecs,
			clusterID:      testClusterID,
			deploymentType: deploymentmetadata.ClassicFullStackDeploymentType,
		},
	}

	instance.Annotations = map[string]string{}
	podSpecs, _ := dsInfo.podSpec()
	require.NotNil(t, podSpecs)
	require.NotEmpty(t, podSpecs.Containers)

	for _, arg := range hostInjectSpecs.Args {
		assert.Contains(t, podSpecs.Containers[0].Args, arg)
	}
	assert.Contains(t, podSpecs.Containers[0].Args, fmt.Sprintf("--set-host-property=OperatorVersion=$(%s)", deploymentmetadata.EnvDtOperatorVersion))

	// deprecated
	t.Run(`has proxy arg`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		podSpecs, _ = dsInfo.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")

		instance.Spec.Proxy = nil
		podSpecs, _ = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	// deprecated
	t.Run(`has proxy arg but feature flag to ignore is enabled`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		instance.Annotations[dynatracev1beta1.AnnotationFeatureOneAgentIgnoreProxy] = "true"
		podSpecs, _ = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	t.Run(`has network zone arg`, func(t *testing.T) {
		instance.Spec.NetworkZone = testValue
		podSpecs, _ = dsInfo.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)

		instance.Spec.NetworkZone = ""
		podSpecs, _ = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)
	})
	t.Run(`has host-id-source arg for classic fullstack`, func(t *testing.T) {
		daemonset, _ := dsInfo.BuildDaemonSet()
		podSpecs = daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=auto")
	})
	t.Run(`has host-id-source arg for hostMonitoring`, func(t *testing.T) {
		hostMonInstance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					HostMonitoring: &dynatracev1beta1.HostInjectSpec{
						Args: []string{testKey, testValue, testUID},
					},
				},
			},
		}

		hostMonInjectSpec := hostMonInstance.Spec.OneAgent.HostMonitoring

		dsInfo := HostMonitoring{
			builderInfo{
				dynakube:       hostMonInstance,
				hostInjectSpec: hostMonInjectSpec,
				clusterID:      testClusterID,
			},
		}
		daemonset, _ := dsInfo.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
	t.Run(`has host-id-source arg for cloudNativeFullstack`, func(t *testing.T) {
		cloudNativeInstance := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
						HostInjectSpec: dynatracev1beta1.HostInjectSpec{Args: []string{testKey, testValue, testUID}},
					},
				},
			},
		}

		dsInfo := HostMonitoring{
			builderInfo{
				dynakube:       cloudNativeInstance,
				hostInjectSpec: &cloudNativeInstance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
				clusterID:      testClusterID,
			},
		}
		daemonset, _ := dsInfo.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
}
