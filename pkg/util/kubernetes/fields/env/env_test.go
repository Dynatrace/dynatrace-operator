package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	testKey1       = "test-key"
	testKey2       = "test-name"
	testAppName    = "dynatrace-operator"
	testAppVersion = "snapshot"
)

func TestFindEnvVar(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: testKey1, Value: testAppVersion},
		{Name: testKey2, Value: testAppName},
	}

	envVar := FindEnvVar(envVars, testKey1)
	assert.NotNil(t, envVar)
	assert.Equal(t, testKey1, envVar.Name)
	assert.Equal(t, testAppVersion, envVar.Value)

	envVar = FindEnvVar(envVars, testKey2)
	assert.NotNil(t, envVar)
	assert.Equal(t, testKey2, envVar.Name)
	assert.Equal(t, testAppName, envVar.Value)

	envVar = FindEnvVar(envVars, "invalid-key")
	assert.Nil(t, envVar)
}

func TestEnvVarIsIn(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: testKey1, Value: testAppVersion},
		{Name: testKey2, Value: testAppName},
	}

	assert.True(t, IsIn(envVars, testKey1))
	assert.True(t, IsIn(envVars, testKey2))
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

func TestDefaultNamespace(t *testing.T) {
	t.Run("Get from env var", func(t *testing.T) {
		testNamespace := "test-namespace"
		t.Setenv(PodNamespace, testNamespace)

		got := DefaultNamespace()
		assert.Equal(t, testNamespace, got)
	})
	t.Run("Get dynatrace", func(t *testing.T) {
		got := DefaultNamespace()
		assert.Equal(t, "dynatrace", got)
	})
}
