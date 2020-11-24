package kubemon

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpullsecret"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testVersion = "1.0.0"
)

func TestBuildImage(t *testing.T) {
	t.Run(`BuildImage with default instance`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		assert.Equal(t, "/linux/activegate", buildImage(instance))
	})
	t.Run(`BuildImage with api url`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testEndpoint + "/api",
			}}
		assert.Equal(t, "test-endpoint/linux/activegate", buildImage(instance))
	})
	t.Run(`BuildImage with api url and activegate version`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				APIURL: testEndpoint + "/api",
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					ActiveGateVersion: testVersion,
				}}}
		assert.Equal(t, "test-endpoint/linux/activegate:"+testVersion, buildImage(instance))
	})
	t.Run(`BuildImage with custom image`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Image: testName,
				}}}
		assert.Equal(t, testName, buildImage(instance))
	})
}

func TestBuildPullSecret(t *testing.T) {
	t.Run(`BuildPullSecret with default instance`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		pullSecret := buildPullSecret(instance)

		assert.NotNil(t, pullSecret)
		assert.Equal(t, corev1.LocalObjectReference{
			Name: dtpullsecret.PullSecretSuffix,
		}, pullSecret)

		instance = &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName},
		}
		pullSecret = buildPullSecret(instance)

		assert.NotNil(t, pullSecret)
		assert.Equal(t, corev1.LocalObjectReference{
			Name: testName + dtpullsecret.PullSecretSuffix,
		}, pullSecret)
	})
	t.Run(`BuildPullSecret with custom pull secret`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				CustomPullSecret: testName,
			}}
		pullSecret := buildPullSecret(instance)

		assert.NotNil(t, pullSecret)
		assert.Equal(t, corev1.LocalObjectReference{
			Name: testName,
		}, pullSecret)
	})
}
