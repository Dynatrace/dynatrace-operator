package token

import (
	"errors"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetScopes(t *testing.T) {
	setupClient := func(t *testing.T, token string, expectErr error, expectScopes []string) *client {
		req := coremock.NewAPIRequest(t)
		req.EXPECT().WithJSONBody(lookupRequest{Token: token}).Return(req).Once()
		req.EXPECT().Execute(new(scopesResponse)).Run(func(obj any) {
			if expectErr == nil {
				target := obj.(*scopesResponse)
				target.Scopes = expectScopes
			}
		}).Return(expectErr).Once()

		client := coremock.NewAPIClient(t)
		client.EXPECT().POST(t.Context(), lookupPath).Return(req).Once()

		return NewClient(client)
	}

	t.Run("success", func(t *testing.T) {
		expectedScopes := []string{"DataExport", "LogExport"}
		client := setupClient(t, "my-token", nil, expectedScopes)

		actualScopes, err := client.GetScopes(t.Context(), "my-token")

		require.NoError(t, err)
		assert.ElementsMatch(t, expectedScopes, actualScopes)
	})

	t.Run("error", func(t *testing.T) {
		client := setupClient(t, "bad-token", errors.New("api error"), nil)

		scopes, err := client.GetScopes(t.Context(), "bad-token")

		require.Error(t, err)
		assert.Nil(t, scopes)
	})
}

func TestScopesResponse_IsEmpty(t *testing.T) {
	// IsEmpty always returns false: an empty scope list is valid and cacheable.
	// It means the token has no scopes — a config error, not a missing response.
	// Cache invalidation happens automatically when the token is updated.
	t.Run("returns false when scopes are set", func(t *testing.T) {
		assert.False(t, (&scopesResponse{Scopes: []string{"DataExport"}}).IsEmpty())
	})

	t.Run("returns false when scopes are empty", func(t *testing.T) {
		assert.False(t, (&scopesResponse{}).IsEmpty())
	})
}
