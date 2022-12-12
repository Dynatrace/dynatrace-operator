package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Run("returns default values when members are nil", func(t *testing.T) {
		dsInfo := builderInfo{
			instance: &dynatracev1beta1.DynaKube{},
		}
		envVars := dsInfo.environmentVariables()

		assert.Contains(t, envVars, corev1.EnvVar{Name: dtClusterId, ValueFrom: nil})
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, dtNodeName))
	})
}
