package resourceattributes

import (
	"net/url"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func Test_newAttributesFromEnv(t *testing.T) {
	t.Run("empty env value returns empty map", func(t *testing.T) {
		attrs := newAttributesFromEnv(corev1.EnvVar{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: ""})
		assert.Empty(t, attrs)
	})

	t.Run("parses key=value pairs and trims whitespace", func(t *testing.T) {
		attrs := newAttributesFromEnv(corev1.EnvVar{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: " k1 = v1 , k2=v2,k3= v3  "})
		require.Len(t, attrs, 3)
		assert.Equal(t, "v1", attrs["k1"])
		assert.Equal(t, "v2", attrs["k2"])
		assert.Equal(t, "v3", attrs["k3"])
	})

	t.Run("ignores malformed entries without '='", func(t *testing.T) {
		attrs := newAttributesFromEnv(corev1.EnvVar{Name: "OTEL_RESOURCE_ATTRIBUTES", Value: "k1=v1,k2,k3=v3"})
		require.Len(t, attrs, 2)
		assert.Contains(t, attrs, "k1")
		assert.Contains(t, attrs, "k3")
		assert.NotContains(t, attrs, "k2")
	})
}

func Test_newAttributesFromMap(t *testing.T) {
	prefix := metadataenrichment.Annotation + "/"
	annotKey := prefix + "service.name"
	annotVal := "my service/value with spaces"
	otherKey := "unrelated.annotation/key"

	input := map[string]string{
		annotKey: annotVal,
		otherKey: "ignore-me",
	}
	attrs := newAttributesFromMap(input)
	require.Len(t, attrs, 1)
	encoded := url.QueryEscape(annotVal)
	assert.Equal(t, encoded, attrs["service.name"])
}

func Test_attributes_merge(t *testing.T) {
	base := attributes{"k1": "v1", "k2": "v2"}
	added := attributes{"k2": "override", "k3": "v3"}
	mutated := base.merge(added)
	assert.True(t, mutated, "expected mutation due to k3 addition")
	// existing value must not be overridden
	assert.Equal(t, "v2", base["k2"]) // ensure no override
	assert.Equal(t, "v3", base["k3"]) // new key added
	// merging only existing keys -> no mutation
	mutatedAgain := base.merge(attributes{"k1": "x", "k2": "y"})
	assert.False(t, mutatedAgain)
}

func Test_attributes_toString(t *testing.T) {
	attrs := attributes{"a": "1", "b": "2", "c": "3"}
	result := attrs.toString()
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

func Test_attributes(t *testing.T) {
	// Build from env then merge annotations
	envAttrs := newAttributesFromEnv(corev1.EnvVar{Value: "k1=v1,k2=v2"})
	annotKey := metadataenrichment.Annotation + "/custom.key"
	annotVal := "value:with/special chars"
	mapAttrs := newAttributesFromMap(map[string]string{annotKey: annotVal})
	mutated := envAttrs.merge(mapAttrs)
	assert.True(t, mutated)
	// ensure encoded value is present
	assert.Equal(t, url.QueryEscape(annotVal), envAttrs["custom.key"])
	// ensure final string contains all keys (order ignored)
	final := envAttrs.toString()
	for _, k := range []string{"k1=v1", "k2=v2", "custom.key=" + url.QueryEscape(annotVal)} {
		assert.Contains(t, final, k)
	}
}
