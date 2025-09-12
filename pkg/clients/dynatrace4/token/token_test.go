package token

import (
	"context"
	"errors"
	"testing"

	coreMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace4/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetTokenScopes(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		expectedScopes := []string{"scope1", "scope2"}
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			// Use type assertion to the anonymous struct type
			if target, ok := args[0].(*struct {
				Scopes []string `json:"scopes"`
			}); ok {
				target.Scopes = expectedScopes
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, mock.Anything).Return(requestBuilder)

		client := &client{apiClient: apiClient}
		scopes, err := client.GetTokenScopes(ctx, "sometoken")
		require.NoError(t, err)
		assert.Equal(t, TokenScopes(expectedScopes), scopes)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("POST", mock.Anything, mock.Anything).Return(requestBuilder)

		client := &client{apiClient: apiClient}
		scopes, err := client.GetTokenScopes(ctx, "sometoken")
		require.Error(t, err)
		assert.Nil(t, scopes)
	})

	t.Run("empty token", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		client := &client{apiClient: apiClient}
		scopes, err := client.GetTokenScopes(ctx, "")
		require.NoError(t, err)
		assert.Nil(t, scopes)
	})

	t.Run("empty scopes returned", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*struct {
				Scopes []string `json:"scopes"`
			}); ok {
				target.Scopes = []string{}
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, mock.Anything).Return(requestBuilder)

		client := &client{apiClient: apiClient}
		scopes, err := client.GetTokenScopes(ctx, "sometoken")
		require.NoError(t, err)
		assert.Empty(t, scopes)
	})
}
