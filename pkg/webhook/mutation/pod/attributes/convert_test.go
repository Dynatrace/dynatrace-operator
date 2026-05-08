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
}
