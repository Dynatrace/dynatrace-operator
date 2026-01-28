package settings

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrUpdateKubernetesSetting(t *testing.T) {
	ctx := t.Context()

	matchBody := func(schemaVersion string) any {
		return matchJSONBody[kubernetesObjectValue](KubernetesSettingsSchemaID, schemaVersion)
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(hierarchicalMonitoringSettingsSchemaVersion)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{"obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(hierarchicalMonitoringSettingsSchemaVersion)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("fallback to v1 on 404", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request2 := coremock.NewAPIRequest(t)

		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(hierarchicalMonitoringSettingsSchemaVersion)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(&core.HTTPError{StatusCode: 404}).Once()

		request2.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request2).Once()
		request2.EXPECT().WithJSONBody(matchBody(schemaVersionV1)).Return(request2).Once()
		request2.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{"obj-456"}})).Return(nil).Once()

		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request2).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-456", objectID)
	})

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "", "scope-1")
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, objectID)
	})

	t.Run("invalid response", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody(hierarchicalMonitoringSettingsSchemaVersion)).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		_, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", "uuid-1", "scope-1")
		require.ErrorAs(t, err, new(notSingleEntryError))
	})
}

func TestCreateOrUpdateKubernetesAppSetting(t *testing.T) {
	ctx := t.Context()

	matchBody := func() any {
		return matchJSONBody[kubernetesAppObjectValue](AppTransitionSchemaID, appTransitionSchemaVersion)
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{"obj-app-1"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, "scope-1")
		require.NoError(t, err)
		assert.Equal(t, "obj-app-1", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, "scope-1")
		require.Error(t, err)
		assert.Empty(t, objectID)
	})
}
