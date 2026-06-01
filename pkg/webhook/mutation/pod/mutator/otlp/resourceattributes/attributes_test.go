package resourceattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestSanitizeValue(t *testing.T) {
	t.Run("empty string is unchanged", func(t *testing.T) {
		assert.Empty(t, sanitizeValue(""))
	})

	t.Run("plain alphanum string is unchanged", func(t *testing.T) {
		assert.Equal(t, "hello", sanitizeValue("hello"))
	})

	envRefCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"K8S_PODNAME (allowlisted) => no changes", "$(K8S_PODNAME)", "$(K8S_PODNAME)"},
		{"K8S_PODUID (allowlisted) => no changes", "$(K8S_PODUID)", "$(K8S_PODUID)"},
		{"K8S_NODE_NAME (allowlisted) => no changes", "$(K8S_NODE_NAME)", "$(K8S_NODE_NAME)"},
		{"DT_API_TOKEN (not allowlisted) => encoded", "$(DT_API_TOKEN)", "%24%28DT_API_TOKEN%29"},
		{"lowercase name (case-sensitive) => encoded", "$(k8s_podname)", "%24%28k8s_podname%29"},
		{"whitespace inside parens => encoded", "$( K8S_PODNAME )", "%24%28+K8S_PODNAME+%29"},
		{"chars before $(...) (not anchored) => encoded", "helloo$(K8S_PODNAME)", "helloo%24%28K8S_PODNAME%29"},
		{"chars after $(...) (not anchored) => encoded", "$(K8S_PODNAME)byeee", "%24%28K8S_PODNAME%29byeee"},
		{"nested reference => encoded", "$($(K8S_PODNAME))", "%24%28%24%28K8S_PODNAME%29%29"},
		{"missing ')' at the end => encoded", "$(K8S_PODNAME", "%24%28K8S_PODNAME"},
		{"unknown name => encoded", "$(UNKNOWN_VAR)", "%24%28UNKNOWN_VAR%29"},
	}
	for _, tc := range envRefCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, sanitizeValue(tc.input))
		})
	}

	t.Run("spaces are encoded as +", func(t *testing.T) {
		assert.Equal(t, "hello+world", sanitizeValue("hello world"))
	})

	t.Run("comma is encoded (would break key=value,key=value format)", func(t *testing.T) {
		assert.Equal(t, "a%2Cb", sanitizeValue("a,b"))
	})

	t.Run("equals sign is encoded (would break key=value format)", func(t *testing.T) {
		assert.Equal(t, "a%3Db", sanitizeValue("a=b"))
	})

	t.Run("already query-encoded value is idempotent", func(t *testing.T) {
		// value that went through a previous sanitizeValue call must not be double-encoded
		once := sanitizeValue("hello world / München")
		twice := sanitizeValue(once)
		assert.Equal(t, once, twice)
	})

	t.Run("value with %XX encoding is normalised and idempotent", func(t *testing.T) {
		// %20 is a valid percent-encoded space; it gets normalised to + (QueryEscape form)
		// and must not become %2520 on a second pass
		normalised := sanitizeValue("hello%20world")
		assert.Equal(t, "hello+world", normalised)
		assert.Equal(t, normalised, sanitizeValue(normalised))
	})

	t.Run("percent sign with invalid sequence is encoded, not double-encoded", func(t *testing.T) {
		// "100%" contains an incomplete percent sequence; url.QueryUnescape returns an error
		// so we fall back to encoding the raw string: % becomes %25
		assert.Equal(t, "100%25", sanitizeValue("100%"))
	})

	t.Run("percent sign mid-string with invalid sequence is encoded", func(t *testing.T) {
		// "50%+off" - the % starts an invalid sequence (%+o is not a valid hex pair)
		assert.Equal(t, "50%25%2Boff", sanitizeValue("50%+off"))
	})

	// Known caveat: a literal + is indistinguishable from a query-encoded space.
	// url.QueryUnescape decodes + as space, so it gets re-encoded as + unchanged.
	// This is acceptable because the OTEL SDK treats both + and %20 as a space anyway.
	t.Run("literal plus sign is treated as an already-encoded space (known caveat)", func(t *testing.T) {
		assert.Equal(t, "a+b", sanitizeValue("a+b"))
	})

	t.Run("already percent-encoded percent sign is idempotent", func(t *testing.T) {
		// "%25" decodes to "%" which re-encodes to "%25" — stable fixed point
		assert.Equal(t, "%25", sanitizeValue("%25"))
		assert.Equal(t, "%25", sanitizeValue(sanitizeValue("%25")))
	})

	t.Run("unicode characters are percent-encoded", func(t *testing.T) {
		encoded := sanitizeValue("München")
		assert.NotEqual(t, "München", encoded)
		// must be idempotent
		assert.Equal(t, encoded, sanitizeValue(encoded))
	})
}

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
