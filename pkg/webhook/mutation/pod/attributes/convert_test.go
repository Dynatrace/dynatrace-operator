package attributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	t.Run("applies convertFunc to each entry and returns all results", func(t *testing.T) {
		attrs := map[string]string{
			"key1": "val1",
			"key2": "val2",
		}
		result := convert(attrs, func(k, v string) string { return k + "=" + v })
		assert.ElementsMatch(t, []string{"key1=val1", "key2=val2"}, result)
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		result := convert(map[string]string{}, func(k, v string) string { return k + "=" + v })
		assert.Empty(t, result)
	})

	t.Run("convertFunc receives both key and value", func(t *testing.T) {
		attrs := map[string]string{"mykey": "myval"}
		var gotKey, gotVal string
		convert(attrs, func(k, v string) string {
			gotKey, gotVal = k, v

			return ""
		})
		assert.Equal(t, "mykey", gotKey)
		assert.Equal(t, "myval", gotVal)
	})

	// Regression: convert() must not include empty strings returned by the convertFunc.
	// If it did, strings.Join(result, ",") would produce spurious commas in
	// OTEL_RESOURCE_ATTRIBUTES (e.g. "k1=v1,,k2=v2") for attributes with empty values
	// such as dt.entity.kubernetes_cluster when KubernetesClusterMEID is not yet set.
	t.Run("empty return values from convertFunc are excluded", func(t *testing.T) {
		attrs := map[string]string{
			"key-with-value": "val",
			"key-empty":      "",
		}
		result := convert(attrs, func(k, v string) string {
			if v == "" {
				return ""
			}

			return k + "=" + v
		})
		assert.Equal(t, []string{"key-with-value=val"}, result)
	})
}
