package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

const (
	expectedBaseInitEnvLen = 7
	expectedBaseEnvLen     = 4
)

func TestGetInitEnvs(t *testing.T) {
	t.Run("get base init envs", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"
		dk.Status.KubeSystemUUID = "test-cluster-uuid"
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"

		envs := getInitEnvs(dk)

		assert.Len(t, envs, expectedBaseInitEnvLen)

		for _, env := range envs {
			hasValueOrFieldPath(t, env)
		}
	})
}

func TestGetEnvs(t *testing.T) {
	t.Run("get base envs", func(t *testing.T) {
		envs := getEnvs()

		assert.Len(t, envs, expectedBaseEnvLen)

		for _, env := range envs {
			hasValueOrFieldPath(t, env)
		}
	})
}

func hasValueOrFieldPath(t *testing.T, env corev1.EnvVar) {
	t.Helper()

	require.NotEmpty(t, env.Name)

	if env.ValueFrom == nil {
		assert.NotEmpty(t, env.Value)
	} else {
		require.NotNil(t, env.ValueFrom.FieldRef)
		assert.NotEmpty(t, env.ValueFrom.FieldRef.FieldPath)
	}
}
