package token

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_String(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected string
	}{
		{
			name:     "nil token returns empty string",
			token:    nil,
			expected: "",
		},
		{
			name:     "token with value returns value",
			token:    &Token{Value: "my-secret-token"},
			expected: "my-secret-token",
		},
		{
			name:     "token with empty value returns empty string",
			token:    &Token{Value: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.token.String())
		})
	}
}

func TestTokenVerifyScopesNoFeatures(t *testing.T) {
	optionalScopes, err := (&Token{}).verifyScopes(t.Context(), nil, dynakube.DynaKube{})
	require.NoError(t, err)
	assert.Empty(t, optionalScopes)
}
