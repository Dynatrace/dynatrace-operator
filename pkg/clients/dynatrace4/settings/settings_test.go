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

func TestGetK8sClusterME(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*getSettingsForKubeSystemUUIDResponse); ok {
				*target = getSettingsForKubeSystemUUIDResponse{
					Settings: []kubernetesSetting{{EntityID: "entity-1", Value: kubernetesSettingValue{Label: "label-1"}}},
				}
			}
		}).Return(nil)
		apiClient.On("GET", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.NoError(t, err)
		assert.Equal(t, K8sClusterME{ID: "entity-1", Name: "label-1"}, me)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Return(errors.New("api error"))
		apiClient.On("GET", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.Error(t, err)
		assert.Equal(t, K8sClusterME{}, me)
	})

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "")
		require.Error(t, err)
		assert.Equal(t, K8sClusterME{}, me)
	})

	t.Run("no settings returned", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*getSettingsForKubeSystemUUIDResponse); ok {
				*target = getSettingsForKubeSystemUUIDResponse{Settings: []kubernetesSetting{}}
			}
		}).Return(nil)
		apiClient.On("GET", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.NoError(t, err)
		assert.Equal(t, K8sClusterME{}, me)
	})
}

func TestGetSettingsForMonitoredEntity(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*GetSettingsResponse); ok {
				*target = GetSettingsResponse{TotalCount: 2}
			}
		}).Return(nil)
		apiClient.On("GET", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := NewClient(apiClient)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterME{ID: "entity-1"}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 2}, resp)
	})

	t.Run("empty monitoredEntity.ID", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		client := NewClient(apiClient)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterME{}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 0}, resp)
	})
}

func TestGetSettingsForLogModule(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		requestBuilder.On("WithQueryParams", mock.Anything).Return(requestBuilder)
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*GetLogMonSettingsResponse); ok {
				*target = GetLogMonSettingsResponse{TotalCount: 3}
			}
		}).Return(nil)
		apiClient.On("GET", mock.Anything, "/v2/settings/objects").Return(requestBuilder)

		client := NewClient(apiClient)
		resp, err := client.GetSettingsForLogModule(ctx, "entity-1")
		require.NoError(t, err)
		assert.Equal(t, GetLogMonSettingsResponse{TotalCount: 3}, resp)
	})

	t.Run("empty monitoredEntity", func(t *testing.T) {
		apiClient := coreMock.NewAPIClient(t)
		client := NewClient(apiClient)
		resp, err := client.GetSettingsForLogModule(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, GetLogMonSettingsResponse{TotalCount: 0}, resp)
	})
}
