// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", K8sClusterRegistration{})
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, objectID)
	})

	t.Run("gen2 uses an empty scope", func(t *testing.T) {
		value := newKubernetesObjectValue("label-1", "uuid-1")

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(KubernetesSettingsSchemaID, hierarchicalMonitoringSettingsSchemaVersion, "", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("fallback to v1 on 404", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request2 := coremock.NewRequest(t)

			v1Value := value
			v1Value.monitoringSettings = &monitoringSettings{CloudApplicationPipelineEnabled: true}

			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(KubernetesSettingsSchemaID, hierarchicalMonitoringSettingsSchemaVersion, "", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(&core.HTTPError{StatusCode: 404}).Once()

			request2.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request2).Once()
			request2.EXPECT().WithJSONBody(matchGen2Body(KubernetesSettingsSchemaID, schemaVersionV1, "", v1Value)).Return(request2).Once()
			request2.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-456"}})).Return(nil).Once()

			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request2).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, "obj-456", objectID)
		})
	})

	t.Run("gen3 uses the cluster-UID lookup scope", func(t *testing.T) {
		value := newKubernetesObjectValue("label-1", "uuid-1")
		const scope = "k8s.cluster.uid:uuid-1"

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(KubernetesSettingsSchemaID, hierarchicalMonitoringSettingsSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Run(injectResponse(postObjectsResponse{ObjectID: "obj-123"})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("fallback to v1 on 404", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request2 := coremock.NewRequest(t)

			v1Value := value
			v1Value.monitoringSettings = &monitoringSettings{CloudApplicationPipelineEnabled: true}

			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(KubernetesSettingsSchemaID, hierarchicalMonitoringSettingsSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Return(&core.HTTPError{StatusCode: 404}).Once()

			request2.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request2).Once()
			request2.EXPECT().WithJSONBody(matchGen3Body(KubernetesSettingsSchemaID, schemaVersionV1, scope, v1Value)).Return(request2).Once()
			request2.EXPECT().Execute(new(postObjectsResponse)).Run(injectResponse(postObjectsResponse{ObjectID: "obj-456"})).Return(nil).Once()

			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request2).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateOrUpdateKubernetesSetting(ctx, "label-1", K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, "obj-456", objectID)
		})
	})
}

func TestCreateOrUpdateKubernetesAppSetting(t *testing.T) {
	ctx := t.Context()
	value := kubernetesAppObjectValue{KubernetesAppOptions: kubernetesAppOptions{EnableKubernetesApp: true}}

	t.Run("gen2 uses the Monitored Entity ID as scope", func(t *testing.T) {
		registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "scope-1"}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(AppTransitionSchemaID, appTransitionSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-app-1"}})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, "obj-app-1", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(AppTransitionSchemaID, appTransitionSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, registration)
			require.Error(t, err)
			assert.Empty(t, objectID)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "scope-1"}
		const scope = "scope-1"

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(AppTransitionSchemaID, appTransitionSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Run(injectResponse(postObjectsResponse{ObjectID: "obj-app-1"})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, "obj-app-1", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(AppTransitionSchemaID, appTransitionSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateOrUpdateKubernetesAppSetting(ctx, registration)
			require.Error(t, err)
			assert.Empty(t, objectID)
		})
	})
}
