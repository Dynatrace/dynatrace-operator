package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
