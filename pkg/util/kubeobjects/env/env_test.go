package env

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestFindEnvVar(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: consts.TestKey1, Value: consts.TestAppVersion},
		{Name: consts.TestKey2, Value: consts.TestAppName},
	}

	envVar := FindEnvVar(envVars, consts.TestKey1)
	assert.NotNil(t, envVar)
	assert.Equal(t, consts.TestKey1, envVar.Name)
	assert.Equal(t, consts.TestAppVersion, envVar.Value)

	envVar = FindEnvVar(envVars, consts.TestKey2)
	assert.NotNil(t, envVar)
	assert.Equal(t, consts.TestKey2, envVar.Name)
	assert.Equal(t, consts.TestAppName, envVar.Value)

	envVar = FindEnvVar(envVars, "invalid-key")
	assert.Nil(t, envVar)
}

func TestEnvVarIsIn(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: consts.TestKey1, Value: consts.TestAppVersion},
		{Name: consts.TestKey2, Value: consts.TestAppName},
	}

	assert.True(t, IsIn(envVars, consts.TestKey1))
	assert.True(t, IsIn(envVars, consts.TestKey2))
	assert.False(t, IsIn(envVars, "invalid-key"))
}

func TestAddOrUpdate(t *testing.T) {
	newEnvVar := corev1.EnvVar{Name: "x", Value: "X"}

	t.Run("Add envvar", func(t *testing.T) {
		envVars := []corev1.EnvVar{
			{Name: "a", Value: "A"},
			{Name: "b", Value: "B"},
		}
		envVars = AddOrUpdate(envVars, newEnvVar)
		assert.Len(t, envVars, 3)
		assert.Contains(t, envVars, newEnvVar)
	})
	t.Run("Update envvar", func(t *testing.T) {
		envVars := []corev1.EnvVar{
			{Name: "a", Value: "A"},
			{Name: "b", Value: "B"},
			{Name: newEnvVar.Name, Value: "this value should be updated"},
		}
		envVars = AddOrUpdate(envVars, newEnvVar)
		assert.Len(t, envVars, 3)
		assert.Contains(t, envVars, newEnvVar)
	})
}
