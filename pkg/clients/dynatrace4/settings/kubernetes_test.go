package settings

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	coreMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace4/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateOrUpdateKubernetesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.MatchedBy(func(arg interface{}) bool {
			bodies, ok := arg.([]postKubernetesSettingsBody)

			return ok && len(bodies) == 1 && bodies[0].SchemaVersion == hierarchicalMonitoringSettingsSchemaVersion
		})).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{{ObjectID: "obj-123"}}
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("fallback to v1 on 404", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)

		// First call: v3 body
		requestBuilder.On("WithJSONBody", mock.MatchedBy(func(arg interface{}) bool {
			bodies, ok := arg.([]postKubernetesSettingsBody)

			return ok && len(bodies) == 1 && bodies[0].SchemaVersion == hierarchicalMonitoringSettingsSchemaVersion
		})).Return(requestBuilder).Once()
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("error: " + strconv.Itoa(http.StatusNotFound))).Once()

		// Second call: v1 body
		requestBuilder.On("WithJSONBody", mock.MatchedBy(func(arg interface{}) bool {
			bodies, ok := arg.([]postKubernetesSettingsBody)

			return ok && len(bodies) == 1 && bodies[0].SchemaVersion == schemaVersionV1
		})).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{{ObjectID: "obj-456"}}
			}
		}).Return(nil)

		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-456", objectID)
	})

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "", "scope-1")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})
}

func TestCreateOrUpdateKubernetesAppSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{{ObjectID: "obj-app-1"}}
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-app-1", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, "scope-1")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})
}

func TestPerformCreateOrUpdateKubernetesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{{ObjectID: "obj-perform-1"}}
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.performCreateOrUpdateKubernetesSetting(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "obj-perform-1", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.performCreateOrUpdateKubernetesSetting(ctx, nil)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("not exactly one entry", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", map[string]string{"validateOnly": "false"}).Return(requestBuilder)
		requestBuilder.On("WithJSONBody", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*[]postSettingsResponse); ok {
				*target = []postSettingsResponse{} // empty response
			}
		}).Return(nil)
		apiClient.On("POST", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := &client{apiClient: apiClient}
		objectID, err := client.performCreateOrUpdateKubernetesSetting(ctx, nil)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})
}
