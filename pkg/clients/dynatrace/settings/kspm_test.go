package settings

import (
	"errors"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetKSPMSettings(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{
		validateOnlyQueryParam: "true",
		schemaIDsQueryParam:    kspmSettingsSchemaID,
		scopesQueryParam:       "entity-1",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(GetSettingsResponse)).Run(injectResponse(GetSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		resp, err := client.GetKSPMSettings(ctx, "entity-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 3}, resp)
	})

	t.Run("empty monitoredEntity", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		resp, err := client.GetKSPMSettings(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 0}, resp)
	})
}

func TestCreateKSPMSetting(t *testing.T) {
	ctx := t.Context()

	matchBody := func() any {
		return matchJSONBody[kspmSettingsValue](kspmSettingsSchemaID, kspmSettingsSchemaVersion)
	}

	t.Run("no ME", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)

		client := NewClient(apiClient)
		objectID, err := client.CreateKSPMSetting(ctx, "", true)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateKSPMSetting(ctx, "scope-1", true)
		require.NoError(t, err)
		assert.Equal(t, "obj-123", objectID)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateKSPMSetting(ctx, "scope-1", true)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("response not exactly one entry", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(nil).Once()
		apiClient.On("POST", ctx, ObjectsPath).Return(request)

		client := NewClient(apiClient)
		objectID, err := client.CreateKSPMSetting(ctx, "scope-1", true)
		require.ErrorAs(t, err, new(tooManyEntriesError))
		assert.Empty(t, objectID)
	})
}
