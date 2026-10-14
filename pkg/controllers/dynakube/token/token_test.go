// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
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
	optionalScopes, err := (&Token{}).verifyScopes(t.Context(), nil, &dynakube.DynaKube{})
	require.NoError(t, err)
	assert.Empty(t, optionalScopes)
}

func TestTokenCollectOptionalScopes(t *testing.T) {
	availableScopes := []string{tokenclient.ScopeSettingsRead, tokenclient.ScopeSettingsWrite}

	t.Run("api token reports all optional scopes even without enabled features", func(t *testing.T) {
		apiToken := newToken(APIKey, "api-token")

		optionalScopes := apiToken.collectOptionalScopes(availableScopes, &dynakube.DynaKube{})

		assert.Equal(t, map[string]OptionalScope{
			tokenclient.ScopeSettingsRead:  {Available: true},
			tokenclient.ScopeSettingsWrite: {Available: true},
		}, optionalScopes)
	})

	t.Run("data ingest token reports no optional scopes", func(t *testing.T) {
		dataIngestToken := newToken(DataIngestKey, "data-ingest-token")

		optionalScopes := dataIngestToken.collectOptionalScopes(availableScopes, &dynakube.DynaKube{})

		assert.Empty(t, optionalScopes)
	})
}
