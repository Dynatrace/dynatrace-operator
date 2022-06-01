package kubeobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestFindEnvVar(t *testing.T) {
	envVars := []v1.EnvVar{
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
	envVars := []v1.EnvVar{
		{Name: testKey1, Value: testAppVersion},
		{Name: testKey2, Value: testAppName},
	}

	assert.True(t, EnvVarIsIn(envVars, testKey1))
	assert.True(t, EnvVarIsIn(envVars, testKey2))
	assert.False(t, EnvVarIsIn(envVars, "invalid-key"))
}
