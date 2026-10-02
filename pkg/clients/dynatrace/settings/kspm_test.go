// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

	t.Run("empty monitoredEntity", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		resp, err := client.GetKSPMSettings(ctx, K8sClusterRegistration{})
		require.NoError(t, err)
		assert.Equal(t, KSPMSettingsResponse{TotalCount: 0}, resp)
	})

	t.Run("gen2", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen2ValidateOnlyParam: "true",
			gen2SchemaIDsParam:    kspmSettingsSchemaID,
			gen2ScopesParam:       "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(KSPMSettingsResponse)).Run(injectResponse(KSPMSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen2ObjectsPath).Return(request).Once()

		client := NewClient(apiClient, Gen2)
		resp, err := client.GetKSPMSettings(ctx, K8sClusterRegistration{EntityScope: "entity-1"})
		require.NoError(t, err)
		assert.Equal(t, KSPMSettingsResponse{TotalCount: 3}, resp)
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen3SchemaIDParam: kspmSettingsSchemaID,
			gen3ScopeParam:    "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(KSPMSettingsResponse)).Run(injectResponse(KSPMSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen3ObjectsPath).Return(request).Once()

		client := NewClient(apiClient, Gen3)
		resp, err := client.GetKSPMSettings(ctx, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "entity-1"})
		require.NoError(t, err)
		assert.Equal(t, KSPMSettingsResponse{TotalCount: 3}, resp)
	})
}

func TestCreateKSPMSetting(t *testing.T) {
	ctx := t.Context()
	value := KSPMSettingsValue{DatasetPipelineEnabled: true}

	t.Run("no ME", func(t *testing.T) {
		apiClient := coremock.NewClient(t)

		client := NewClient(apiClient, Gen2)
		objectID, err := client.CreateKSPMSetting(ctx, K8sClusterRegistration{}, true)
		require.Error(t, err)
		assert.Empty(t, objectID)
	})

	t.Run("gen2", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(kspmSettingsSchemaID, kspmSettingsSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			settingsClient := NewClient(apiClient, Gen2)
			objectID, err := settingsClient.CreateKSPMSetting(ctx, K8sClusterRegistration{EntityScope: "scope-1"}, true)
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(kspmSettingsSchemaID, kspmSettingsSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateKSPMSetting(ctx, K8sClusterRegistration{EntityScope: "scope-1"}, true)
			require.Error(t, err)
			assert.Empty(t, objectID)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(kspmSettingsSchemaID, kspmSettingsSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Run(injectResponse(postObjectsResponse{ObjectID: "obj-123"})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			settingsClient := NewClient(apiClient, Gen3)
			objectID, err := settingsClient.CreateKSPMSetting(ctx, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "scope-1"}, true)
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(kspmSettingsSchemaID, kspmSettingsSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateKSPMSetting(ctx, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "scope-1"}, true)
			require.Error(t, err)
			assert.Empty(t, objectID)
		})
	})
}
