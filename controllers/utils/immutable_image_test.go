package utils

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testName = "test-name"

func TestBuildPullSecret(t *testing.T) {
	t.Run(`BuildPullSecret with default instance`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{}
		pullSecret := BuildPullSecret(instance)

		assert.NotNil(t, pullSecret)
		assert.Equal(t, corev1.LocalObjectReference{
			Name: dtpullsecret.PullSecretSuffix,
		}, pullSecret)

		instance = &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName},
		}
		pullSecret = BuildPullSecret(instance)

		assert.Equal(t, corev1.LocalObjectReference{
			Name: testName + dtpullsecret.PullSecretSuffix,
		}, pullSecret)
	})
	t.Run(`BuildPullSecret with custom pull secret`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CustomPullSecret: testName,
			}}
		pullSecret := BuildPullSecret(instance)

		assert.NotNil(t, pullSecret)
		assert.Equal(t, corev1.LocalObjectReference{
			Name: testName,
		}, pullSecret)
	})
}
