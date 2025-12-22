package resourceattributes

import (
	"net/url"
	"strings"
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

func TestSanitizeMap(t *testing.T) {
	annotKey := "service.name"
	annotVal := "my service/value with spaces"

	input := map[string]string{
		annotKey: annotVal,
	}
	attrs := sanitizeMap(input)
	require.Len(t, attrs, 1)
	encoded := url.QueryEscape(annotVal)
	assert.Equal(t, encoded, attrs["service.name"])
}

func TestSanitizeMapPreservesEnvRefsAndEncodesValues(t *testing.T) {
	input := map[string]string{
		"k8s.pod.name":     "$(K8S_PODNAME)",
		"k8s.cluster.name": "bh-eks-test1 with space=equals,comma",
	}

	sanitized := sanitizeMap(input)

	assert.Equal(t, "$(K8S_PODNAME)", sanitized["k8s.pod.name"])

	expected := url.QueryEscape("bh-eks-test1 with space=equals,comma")
	assert.Equal(t, expected, sanitized["k8s.cluster.name"])
}

func TestAttributesMerge(t *testing.T) {
	base := Attributes{"k1": "v1", "k2": "v2"}
	added := Attributes{"k2": "override", "k3": "v3"}
	mutated := base.Merge(added)
	assert.True(t, mutated, "expected mutation due to k3 addition")
	// existing value must not be overridden
	assert.Equal(t, "v2", base["k2"]) // ensure no override
	assert.Equal(t, "v3", base["k3"]) // new key added
	// merging only existing keys -> no mutation
	mutatedAgain := base.Merge(Attributes{"k1": "x", "k2": "y"})
	assert.False(t, mutatedAgain)
}

func TestAttributesToString(t *testing.T) {
	attrs := Attributes{"a": "1", "b": "2", "c": "3", "keyWithEmptyValue": "", "": "valueWithEmptyKey"}
	result := attrs.String()
	parts := strings.Split(result, ",")
	require.Len(t, parts, 3)
	pairs := map[string]string{}
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		require.Len(t, kv, 2)
		pairs[kv[0]] = kv[1]
	}
	assert.Equal(t, "1", pairs["a"])
	assert.Equal(t, "2", pairs["b"])
	assert.Equal(t, "3", pairs["c"])
}

func TestAttributes(t *testing.T) {
	const envName = "OTEL_RESOURCE_ATTRIBUTES"
	// Build from env then Merge annotations
	envAttrs, found := NewAttributesFromEnv([]corev1.EnvVar{{Name: envName, Value: "k1=v1,k2=v2"}}, envName)
	require.True(t, found)
	annotKey := "custom.key"
	annotVal := "value:with/special chars"
	mapAttrs := sanitizeMap(map[string]string{annotKey: annotVal})
	mutated := envAttrs.Merge(mapAttrs)
	assert.True(t, mutated)
	// ensure encoded value is present
	assert.Equal(t, url.QueryEscape(annotVal), envAttrs["custom.key"])
	// ensure final string contains all keys (order ignored)
	final := envAttrs.String()
	for _, k := range []string{"k1=v1", "k2=v2", "custom.key=" + url.QueryEscape(annotVal)} {
		assert.Contains(t, final, k)
	}
}
