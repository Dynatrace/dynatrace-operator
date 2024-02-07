//go:build e2e

package rand

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected string
	}{
		{[]byte{1, 2, 3}, "bcd"},
		{[]byte{255, 255, 255}, "ddd"},
		{[]byte{0, 0, 255}, "aad"},
	}

	for _, tc := range testCases {
		actual := encode(tc.input)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestGetNetworkZoneName(t *testing.T) {
	const prefix = "op-e2e-"

	name, err := GetRandomName(WithLength(defaultLength), WithPrefix("op-e2e-"))
	require.NoError(t, err)

	assert.Len(t, name, len(prefix)+defaultLength)
	assert.True(t, strings.HasPrefix(name, "op-e2e-"))

	name, found := strings.CutPrefix(name, prefix)
	require.True(t, found)

	for _, r := range name {
		assert.Contains(t, "abcdefghijklmnopqrstuvwxyz0123456789", string(r))
	}
}
