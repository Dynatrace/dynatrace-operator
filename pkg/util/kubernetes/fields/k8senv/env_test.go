package k8senv

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

	envVar := Find(envVars, testKey1)
	assert.NotNil(t, envVar)
	assert.Equal(t, testKey1, envVar.Name)
	assert.Equal(t, testAppVersion, envVar.Value)

	envVar = Find(envVars, testKey2)
	assert.NotNil(t, envVar)
	assert.Equal(t, testKey2, envVar.Name)
	assert.Equal(t, testAppName, envVar.Value)

	envVar = Find(envVars, "invalid-key")
	assert.Nil(t, envVar)
}

func TestEnvVarIsIn(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: testKey1, Value: testAppVersion},
		{Name: testKey2, Value: testAppName},
	}

	assert.True(t, Contains(envVars, testKey1))
	assert.True(t, Contains(envVars, testKey2))
	assert.False(t, Contains(envVars, "invalid-key"))
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

func TestAppend(t *testing.T) {
	t.Run("append new", func(t *testing.T) {
		envVars := []corev1.EnvVar{{Name: "a", Value: "A"}}
		envVars, added := Append(envVars, corev1.EnvVar{Name: "b", Value: "B"})
		assert.True(t, added)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "a", Value: "A"},
			{Name: "b", Value: "B"},
		}, envVars)
	})

	t.Run("skip existing", func(t *testing.T) {
		envVars := []corev1.EnvVar{{Name: "a", Value: "A"}}
		envVars, added := Append(envVars, corev1.EnvVar{Name: "a", Value: "X"})
		assert.False(t, added)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "a", Value: "A"},
		}, envVars)
	})

	t.Run("append to empty", func(t *testing.T) {
		var envVars []corev1.EnvVar
		envVars, added := Append(envVars, corev1.EnvVar{Name: "z", Value: "Z"})
		assert.True(t, added)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "z", Value: "Z"},
		}, envVars)
	})

	t.Run("append multiple existing", func(t *testing.T) {
		envVars := []corev1.EnvVar{{Name: "a", Value: "A"}, {Name: "b", Value: "B"}}
		envVars, added := Append(envVars, corev1.EnvVar{Name: "c", Value: "C"})
		assert.True(t, added)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "a", Value: "A"},
			{Name: "b", Value: "B"},
			{Name: "c", Value: "C"},
		}, envVars)
	})
}

func TestFindEnvVarCaseInsensitive(t *testing.T) {
	type args struct {
		envVars []corev1.EnvVar
		name    string
	}
	tests := []struct {
		name string
		args args
		want *corev1.EnvVar
	}{
		{
			name: "found with same case",
			args: args{
				envVars: []corev1.EnvVar{
					{Name: "KEY_ONE", Value: "value1"},
					{Name: "key_two", Value: "value2"},
				},
				name: "KEY_ONE",
			},
			want: &corev1.EnvVar{Name: "KEY_ONE", Value: "value1"},
		},
		{
			name: "found with different case",
			args: args{
				envVars: []corev1.EnvVar{
					{Name: "KEY_ONE", Value: "value1"},
					{Name: "key_two", Value: "value2"},
				},
				name: "key_one",
			},
			want: &corev1.EnvVar{Name: "KEY_ONE", Value: "value1"},
		},
		{
			name: "not found",
			args: args{
				envVars: []corev1.EnvVar{
					{Name: "KEY_ONE", Value: "value1"},
					{Name: "key_two", Value: "value2"},
				},
				name: "key_three",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, FindCaseInsensitive(tt.args.envVars, tt.args.name), "FindCaseInsensitive(%v, %v)", tt.args.envVars, tt.args.name)
		})
	}
}
