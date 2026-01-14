package settings

import (
	"errors"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetK8sClusterME(t *testing.T) {
	ctx := t.Context()
	params := map[string]string{
		validateOnlyQueryParam: "true",
		pageSizeQueryParam:     entitiesPageSize,
		schemaIDsQueryParam:    kubernetesSettingsSchemaID,
		fieldsQueryParam:       kubernetesSettingsNeededFields,
		filterQueryParam:       "value.clusterId='uuid-1'",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Run(func(obj any) {
			target := obj.(*getKubernetesObjectsResponse)
			target.Items = []kubernetesObject{
				{Scope: "entity-1", Value: kubernetesObjectValue{Label: "label-1"}},
			}
		}).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.NoError(t, err)
		assert.Equal(t, K8sClusterME{ID: "entity-1", Name: "label-1"}, me)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.Error(t, err)
		assert.Equal(t, K8sClusterME{}, me)
	})

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "")
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Equal(t, K8sClusterME{}, me)
	})

	t.Run("no settings returned", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		me, err := client.GetK8sClusterME(ctx, "uuid-1")
		require.NoError(t, err)
		assert.Equal(t, K8sClusterME{}, me)
	})
}

func TestGetSettingsForMonitoredEntity(t *testing.T) {
	ctx := t.Context()

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    "schema-1",
			scopesQueryParam:       "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(GetSettingsResponse)).Run(func(obj any) {
			target := obj.(*GetSettingsResponse)
			target.TotalCount = 2
		}).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request)

		client := NewClient(apiClient)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterME{ID: "entity-1"}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 2}, resp)
	})

	t.Run("empty monitoredEntity.ID", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterME{}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 0}, resp)
	})
}

func injectResponse[T any](resp T) func(any) {
	return func(arg any) {
		if target, ok := arg.(*T); ok {
			*target = resp
		}
	}
}

func matchJSONBody[T any](schemaID, schemaVersion string) any {
	// MatchedBy returns an unexported type
	return mock.MatchedBy(func(arg any) bool {
		body, ok := arg.([]postObjectsBody[T])

		return ok && len(body) == 1 && schemaID == body[0].SchemaID && schemaVersion == body[0].SchemaVersion
	})
}
