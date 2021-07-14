package oneagent

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUID = "test-uid"

	testContainerImageVersion = "1.203.0.20200908-220956"

	testKey   = "test-key"
	testValue = "test-value"
)

func TestNewPodSpecForCR_Arguments(t *testing.T) {
	log := logger.NewDTLogger()
	instance := &dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				Args: []string{testKey, testValue, testUID},
			}},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				VersionStatus: dynatracev1alpha1.VersionStatus{
					Version: testContainerImageVersion,
				},
			},
		}}
	metadata := deploymentmetadata.NewDeploymentMetadata(testUID, *instance)
	fullStackSpecs := &instance.Spec.ClassicFullStack
	podSpecs := newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
	require.NotNil(t, podSpecs)
	require.NotEmpty(t, podSpecs.Containers)

	for _, arg := range fullStackSpecs.Args {
		assert.Contains(t, podSpecs.Containers[0].Args, arg)
	}
	assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-property=OperatorVersion="+version.Version)

	metadataArgs := metadata.AsArgs()
	for _, metadataArg := range metadataArgs {
		assert.Contains(t, podSpecs.Containers[0].Args, metadataArg)
	}

	t.Run(`has proxy arg`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1alpha1.DynaKubeProxy{Value: testValue}
		podSpecs := newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")

		instance.Spec.Proxy = nil
		podSpecs = newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-proxy=$(https_proxy)")
	})
	t.Run(`has network zone arg`, func(t *testing.T) {
		instance.Spec.NetworkZone = testValue
		podSpecs := newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)

		instance.Spec.NetworkZone = ""
		podSpecs = newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
		assert.NotContains(t, podSpecs.Containers[0].Args, "--set-network-zone="+testValue)
	})
	t.Run(`has webhook injection arg`, func(t *testing.T) {
		podSpecs = newPodSpecForCR(instance, fullStackSpecs, InframonFeature, true, log, testUID)
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=k8s-node-name")

		podSpecs = newPodSpecForCR(instance, fullStackSpecs, ClassicFeature, true, log, testUID)
		assert.Contains(t, podSpecs.Containers[0].Args, "--set-host-id-source=auto")
	})
}
