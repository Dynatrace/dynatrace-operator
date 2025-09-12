package settings

import (
	"context"
	"errors"
	"testing"

	coreMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace4/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateLogMonitoringSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParam", "validateOnly", "false").Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{{ObjectID: "obj-123"}}
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParam", "validateOnly", "false").Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("response not exactly one entry", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParam", "validateOnly", "false").Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{} // empty response
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})
}
