package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	expectedBaseEnvLen = 4
)

func TestGetEnvs(t *testing.T) {
	t.Run("get base envs", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"
		tenant := "test-tenant"
		envs := getEnvs(dk, tenant)

		assert.Len(t, envs, expectedBaseEnvLen)

		for _, env := range envs {
			hasValueOrRef(t, env)
		}
	})

	t.Run("adds cert env", func(t *testing.T) {
		dk := getDynaKubeWithCerts(t)
		dk.Name = "dk-name-test"
		tenant := "test-tenant"
		envs := getEnvs(dk, tenant)

		assert.Len(t, envs, expectedBaseEnvLen+1)

		for _, env := range envs {
			hasValueOrRef(t, env)
		}
	})

	t.Run("adds user env", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"
		tenant := "test-tenant"
		dk.KSPM().Env = []corev1.EnvVar{
			{Name: "env1", Value: "value1"},
			{Name: "env2", Value: "value2"},
		}

		envs := getEnvs(dk, tenant)

		assert.Len(t, envs, expectedBaseEnvLen+len(dk.KSPM().Env))

		for _, env := range envs {
			hasValueOrRef(t, env)
		}

		for _, env := range dk.KSPM().Env {
			assert.Contains(t, envs, env)
		}
	})
}

func hasValueOrRef(t *testing.T, env corev1.EnvVar) {
	t.Helper()

	require.NotEmpty(t, env.Name)

	if env.ValueFrom == nil {
		assert.NotEmpty(t, env.Value)
	} else {
		switch {
		case env.ValueFrom.FieldRef != nil:
			assert.NotEmpty(t, env.ValueFrom.FieldRef.FieldPath)
		case env.ValueFrom.SecretKeyRef != nil:
			assert.NotEmpty(t, env.ValueFrom.SecretKeyRef.LocalObjectReference)
			assert.NotEmpty(t, env.ValueFrom.SecretKeyRef.Key)
		default:
			t.Fail()
		}
	}
}
