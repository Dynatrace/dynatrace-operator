package resourceattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestNewAttributesFromEnv(t *testing.T) {
	const envName = "OTEL_RESOURCE_ATTRIBUTES"

	t.Run("env var missing returns empty map and found=false", func(t *testing.T) {
		attrs, found := NewAttributesFromEnv([]corev1.EnvVar{}, envName)
		assert.False(t, found)
		assert.Empty(t, attrs)
	})

	t.Run("empty env value returns empty map and found=true", func(t *testing.T) {
		attrs, found := NewAttributesFromEnv([]corev1.EnvVar{{Name: envName, Value: ""}}, envName)
		assert.True(t, found)
		assert.Empty(t, attrs)
	})

	t.Run("parses key=value pairs and trims whitespace", func(t *testing.T) {
		attrs, found := NewAttributesFromEnv([]corev1.EnvVar{{Name: envName, Value: " k1 = v1 , k2=v2,k3= v3  "}}, envName)
		require.True(t, found)
		require.Len(t, attrs, 3)
		assert.Equal(t, "v1", attrs["k1"])
		assert.Equal(t, "v2", attrs["k2"])
		assert.Equal(t, "v3", attrs["k3"])
	})

	t.Run("ignores malformed entries without '=' or without value", func(t *testing.T) {
		attrs, found := NewAttributesFromEnv([]corev1.EnvVar{{Name: envName, Value: "k1=v1,k2,k3=v3"}}, envName)
		require.True(t, found)
		require.Len(t, attrs, 2)
		assert.Contains(t, attrs, "k1")
		assert.Contains(t, attrs, "k3")
		assert.NotContains(t, attrs, "k2")
		assert.NotContains(t, attrs, "k4")
	})
}
