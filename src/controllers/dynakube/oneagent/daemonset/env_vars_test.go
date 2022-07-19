package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Run("returns default values when members are nil", func(t *testing.T) {
		dsInfo := builderInfo{}
		envVars := dsInfo.environmentVariables()

		assert.Contains(t, envVars, corev1.EnvVar{Name: dtClusterId, ValueFrom: nil})
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, dtNodeName))
	})
}
