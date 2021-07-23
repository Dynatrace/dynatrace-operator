package oneagent

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	testUID = "test-uid"

	testContainerImageVersion = "1.203.0.20200908-220956"

	testKey   = "test-key"
	testValue = "test-value"

	testReadOnlyInstallationVolumePath = "/my/path/to/a/directory"
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
	metadata := deploymentmetadata.NewDeploymentMetadata(testUID)
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

func TestEnvVars(t *testing.T) {
	log := logger.NewDTLogger()
	reservedVariable := "DT_K8S_NODE_NAME"
	instance := dynatracev1alpha1.DynaKube{
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL:   testURL,
			OneAgent: dynatracev1alpha1.OneAgentSpec{},
			ClassicFullStack: dynatracev1alpha1.FullStackSpec{
				UseImmutableImage: true,
				Env: []corev1.EnvVar{
					{
						Name:  testName,
						Value: testValue,
					},
					{
						Name:  reservedVariable,
						Value: testValue,
					},
				},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	podSpecs := newPodSpecForCR(&instance, &instance.Spec.ClassicFullStack, ClassicFeature, true, log, testClusterID)
	assert.NotNil(t, podSpecs)
	assert.NotEmpty(t, podSpecs.Containers)
	assert.NotEmpty(t, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, testName, testValue, podSpecs.Containers[0].Env)
	assertHasEnvVar(t, reservedVariable, testValue, podSpecs.Containers[0].Env)
}

func TestNewPodSpecForCR_ReadOnlyMode(t *testing.T) {
	instance := &dynatracev1alpha1.DynaKube{}
	fullstackSpec := &dynatracev1alpha1.FullStackSpec{
		ReadOnly: dynatracev1alpha1.ReadOnlySpec{
			Enabled: true,
			InstallationVolume: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: testReadOnlyInstallationVolumePath,
				},
			},
		},
	}
	podSpec := newPodSpecForCR(instance, fullstackSpec, "", true, logger.NewDTLogger(), "")

	assert.NotNil(t, podSpec)
	assert.Contains(t, podSpec.Volumes, corev1.Volume{
		Name: oneagentInstallationMountName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: testReadOnlyInstallationVolumePath,
				Type: nil,
			},
		},
	})
	assert.Contains(t, podSpec.Containers[0].Env, corev1.EnvVar{
		Name:  oneagentReadOnlyMode,
		Value: "true",
	})

	oneagentInstallationMountExists := false
	for _, volumeMount := range podSpec.Containers[0].VolumeMounts {
		if volumeMount.Name == hostRootMount {
			assert.True(t, volumeMount.ReadOnly)
		}
		if volumeMount.Name == oneagentInstallationMountName {
			assert.Equal(t, oneagentInstallationMountPath, volumeMount.MountPath)
			oneagentInstallationMountExists = true
		}
	}

	assert.True(t, oneagentInstallationMountExists)
}

func assertHasEnvVar(t *testing.T, expectedName string, expectedValue string, envVars []corev1.EnvVar) {
	hasVariable := false
	for _, env := range envVars {
		if env.Name == expectedName {
			hasVariable = true
			assert.Equal(t, expectedValue, env.Value)
		}
	}
	assert.True(t, hasVariable)
}
