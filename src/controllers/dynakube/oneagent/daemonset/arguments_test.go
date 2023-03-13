package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUID                   = "test-uid"
	testContainerImageVersion = "1.203.0.20200908-220956"

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
		arguments := builder.arguments()
		expectedDefaultArguments := builder.appendImmutableImageArgs(appendOperatorVersionArg([]string{}))

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
		podSpecs := dsInfo.podSpec()
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)
		assert.Contains(t, podSpecs.Containers[0].Args, testValue)
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
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				VersionStatus: dynatracev1beta1.VersionStatus{
					Version: testContainerImageVersion,
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
	podSpecs := dsInfo.podSpec()
	require.NotNil(t, podSpecs)
	require.NotEmpty(t, podSpecs.Containers)

	for _, arg := range hostInjectSpecs.Args {
		assert.Contains(t, podSpecs.Containers[0].Args, arg)
	}
	assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-property=OperatorVersion="+version.Version)

	t.Run(`has proxy arg`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		podSpecs = dsInfo.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")

		instance.Spec.Proxy = nil
		podSpecs = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	t.Run(`has proxy arg but feature flag to ignore is enabled`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		instance.Annotations[dynatracev1beta1.AnnotationFeatureOneAgentIgnoreProxy] = "true"
		podSpecs = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	t.Run(`has network zone arg`, func(t *testing.T) {
		instance.Spec.NetworkZone = testValue
		podSpecs = dsInfo.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)

		instance.Spec.NetworkZone = ""
		podSpecs = dsInfo.podSpec()
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)
	})
	t.Run(`has webhook injection arg`, func(t *testing.T) {
		daemonset, _ := dsInfo.BuildDaemonSet()
		podSpecs = daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=auto")

		dsInfo := HostMonitoring{
			builderInfo{
				dynakube:       instance,
				hostInjectSpec: hostInjectSpecs,
				clusterID:      testClusterID,
			},
		}
		daemonset, _ = dsInfo.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
}
