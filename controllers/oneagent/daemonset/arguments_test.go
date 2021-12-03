package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/version"
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
	instance := dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testURL,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{
						Args: []string{testValue},
					},
				},
			},
		},
	}
	dsInfo := ClassicFullStack{
		builderInfo{
			instance:       &instance,
			hostInjectSpec: &instance.Spec.OneAgent.ClassicFullStack.HostInjectSpec,
			clusterId:      testClusterID,
			relatedImage:   testValue,
		},
	}
	podSpecs := dsInfo.podSpec()
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.Contains(t, podSpecs.Containers[0].Args, testValue)
}

func TestPodSpec_Arguments(t *testing.T) {
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.ClassicFullStackSpec{
					HostInjectSpec: dynatracev1beta1.HostInjectSpec{
						Args: []string{testKey, testValue, testUID},
					},
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
	metadata := deploymentmetadata.NewDeploymentMetadata(testClusterID, deploymentmetadata.DeploymentTypeFullStack)
	hostInjectSpecs := &instance.Spec.OneAgent.ClassicFullStack.HostInjectSpec
	dsInfo := ClassicFullStack{
		builderInfo{
			instance:       instance,
			hostInjectSpec: hostInjectSpecs,
			clusterId:      testClusterID,
			relatedImage:   testValue,
			deploymentType: deploymentmetadata.DeploymentTypeFullStack,
		},
	}

	podSpecs := dsInfo.podSpec()
	require.NotNil(t, podSpecs)
	require.NotEmpty(t, podSpecs.Containers)

	for _, arg := range hostInjectSpecs.Args {
		assert.Contains(t, podSpecs.Containers[0].Args, arg)
	}
	assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-property=OperatorVersion="+version.Version)

	metadataArgs := metadata.AsArgs()
	for _, metadataArg := range metadataArgs {
		assert.Contains(t, podSpecs.Containers[0].Args, metadataArg)
	}

	t.Run(`has proxy arg`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		podSpecs = dsInfo.podSpec()
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")

		instance.Spec.Proxy = nil
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
				instance:       instance,
				hostInjectSpec: hostInjectSpecs,
				clusterId:      testClusterID,
				relatedImage:   testValue,
			},
			HostMonitoringFeature,
		}
		daemonset, _ = dsInfo.BuildDaemonSet()
		podSpecs := daemonset.Spec.Template.Spec
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")
	})
}
