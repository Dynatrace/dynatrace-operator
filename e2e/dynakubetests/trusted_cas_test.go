// +build e2e

package dynakubetests

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Fails since Operator does not create OneAgent pods when certs are invalid
func TestTrustedCAs(t *testing.T) {
	apiURL, clt := prepareDefaultEnvironment(t)
	oneAgent := createMinimumViableOneAgent(apiURL)

	trustedCAs := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: namespace,
		},
		Data: map[string]string{testCertName: testData},
	}
	oneAgent.Spec.TrustedCAs = testName

	// prevent creation of pull secret, which would fail due to the test cert being invalid
	oneAgent.Spec.CustomPullSecret = testName

	err := clt.Create(context.TODO(), &trustedCAs)
	require.NoError(t, err)

	err = clt.Create(context.TODO(), &oneAgent)
	assert.NoError(t, err)

	phaseWait := e2e.NewOneAgentWaitConfiguration(t, clt, maxWaitCycles, namespace, testName)
	// Waiting for error, since given certificate is not actually a valid certificate
	err = phaseWait.WaitForPhase(v1alpha1.Error)
	assert.NoError(t, err)

	_, pods := findOneAgentPods(t, clt)
	assert.NotEmpty(t, pods.Items)

	for _, pod := range pods.Items {
		assert.NotEmpty(t, pod.Spec.Volumes)
		assert.NotEmpty(t, pod.Spec.Containers)
		checkVolumes(t, pod.Spec.Volumes)
		checkContainerVolumeMounts(t, pod.Spec.Containers)
	}
}

func checkContainerVolumeMounts(t *testing.T, containers []v1.Container) {
	for _, container := range containers {
		assert.NotEmpty(t, container.VolumeMounts)
		checkVolumeMounts(t, container.VolumeMounts)
	}
}

func checkVolumeMounts(t *testing.T, mounts []v1.VolumeMount) {
	for _, volumeMount := range mounts {
		if volumeMount.Name == testCertName {
			assert.Equal(t, trustedCertPath, volumeMount.MountPath)
			break
		}
	}
}

func checkVolumes(t *testing.T, volumes []v1.Volume) {
	for _, volume := range volumes {
		if volume.ConfigMap != nil && volume.ConfigMap.Name == testName {
			assert.NotEmpty(t, volume.ConfigMap.Items)
			checkConfigMapVolumeItems(t, volume.ConfigMap.Items)
			break
		}
	}
}

func checkConfigMapVolumeItems(t *testing.T, items []v1.KeyToPath) {
	for _, item := range items {
		if item.Key == testCertName {
			assert.Equal(t, trustedCertFilename, item.Path)
			break
		}
	}
}
