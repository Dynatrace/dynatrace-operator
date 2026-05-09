package resourceattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeResourceAttributes(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		override map[string]string
		expected map[string]string
	}{
		{
			name:     "only base set returns copy of base",
			base:     map[string]string{"a": "1", "b": "2"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "only override set returns copy of override",
			override: map[string]string{"x": "10", "y": "20"},
			expected: map[string]string{"x": "10", "y": "20"},
		},
		{
			name:     "both set with no overlap returns union",
			base:     map[string]string{"a": "1"},
			override: map[string]string{"b": "2"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "both set with overlapping keys override wins",
			base:     map[string]string{"a": "1", "shared": "base"},
			override: map[string]string{"b": "2", "shared": "override"},
			expected: map[string]string{"a": "1", "b": "2", "shared": "override"},
		},
		{
			name:     "both nil returns nil",
			expected: nil,
		},
		{
			name:     "both empty returns nil",
			base:     map[string]string{},
			override: map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Merge(tt.base, tt.override))
		})
	}
}

func TestMergeResourceAttributes_ReturnedMapIsCopy(t *testing.T) {
	base := map[string]string{"a": "1"}
	override := map[string]string{"b": "2"}
	result := Merge(base, override)
	require.NotNil(t, result)

	result["new-key"] = "new-value"
	assert.NotContains(t, base, "new-key")
	assert.NotContains(t, override, "new-key")
}
