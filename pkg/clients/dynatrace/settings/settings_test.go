// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"errors"
	"reflect"
	"testing"

	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetK8sClusterME(t *testing.T) {
	ctx := t.Context()

	t.Run("empty kubeSystemUUID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{})
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Equal(t, K8sClusterRegistration{}, me)
	})

	t.Run("gen2", func(t *testing.T) {
		params := map[string]string{
			gen2ValidateOnlyParam: "true",
			gen2PageSizeParam:     entitiesPageSize,
			gen2SchemaIDsParam:    KubernetesSettingsSchemaID,
			gen2FieldsParam:       kubernetesSettingsNeededFields,
			gen2FilterParam:       "value.clusterId='uuid-1'",
		}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Run(func(obj any) {
				target := obj.(*getKubernetesObjectsResponse)
				target.Items = []kubernetesObject{
					{Scope: "entity-1", Value: kubernetesObjectValue{Label: "label-1"}},
				}
			}).Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "entity-1", EntityLabel: "label-1"}, me)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.Error(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1"}, me)
		})

		t.Run("no settings returned", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1"}, me)
		})
	})

	t.Run("gen3", func(t *testing.T) {
		params := map[string]string{
			gen3SchemaIDParam:  KubernetesSettingsSchemaID,
			gen3ScopeParam:     "k8s.cluster.uid:uuid-1",
			gen3AddFieldsParam: kubernetesSettingsNeededFields,
		}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Run(func(obj any) {
				target := obj.(*getKubernetesObjectsResponse)
				target.Items = []kubernetesObject{
					{Scope: "KUBERNETES_CLUSTER-123", Value: kubernetesObjectValue{Label: "label-1"}},
				}
			}).Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123", EntityLabel: "label-1"}, me)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().GET(anyCtx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.Error(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1"}, me)
		})

		t.Run("no settings returned", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(getKubernetesObjectsResponse)).Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			me, err := client.GetK8sClusterME(ctx, K8sClusterRegistration{EntityID: "uuid-1"})
			require.NoError(t, err)
			assert.Equal(t, K8sClusterRegistration{EntityID: "uuid-1"}, me)
		})
	})
}

func TestGetSettingsForMonitoredEntity(t *testing.T) {
	ctx := t.Context()

	t.Run("empty monitoredEntity.ID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterRegistration{}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 0}, resp)
	})

	t.Run("gen2", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen2ValidateOnlyParam: "true",
			gen2SchemaIDsParam:    "schema-1",
			gen2ScopesParam:       "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(TotalCountSettingsResponse)).Run(injectResponse(TotalCountSettingsResponse{TotalCount: 2})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen2ObjectsPath).Return(request)

		client := NewClient(apiClient, Gen2)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterRegistration{EntityScope: "entity-1"}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 2}, resp)
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen3SchemaIDParam: "schema-1",
			gen3ScopeParam:    "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(TotalCountSettingsResponse)).Run(injectResponse(TotalCountSettingsResponse{TotalCount: 2})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen3ObjectsPath).Return(request)

		client := NewClient(apiClient, Gen3)
		resp, err := client.GetSettingsForMonitoredEntity(ctx, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "entity-1"}, "schema-1")
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 2}, resp)
	})
}

func TestDeleteSettings(t *testing.T) {
	ctx := t.Context()
	objectID := "settings-object-123"

	t.Run("empty objectID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		err := client.DeleteSettings(ctx, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, errNoSettingsIDProvided)
	})

	t.Run("gen2", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().Execute(nil).Return(nil).Once()
			apiClient.EXPECT().DELETE(ctx, "/v2/settings/objects/"+objectID).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			err := client.DeleteSettings(ctx, objectID)
			require.NoError(t, err)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().Execute(nil).Return(errors.New("api error")).Once()
			apiClient.EXPECT().DELETE(ctx, "/v2/settings/objects/"+objectID).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			err := client.DeleteSettings(ctx, objectID)
			require.Error(t, err)
			assert.ErrorIs(t, err, errDeleteSettings)
		})
	})

	t.Run("gen3", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			getRequest := coremock.NewRequest(t)
			getRequest.EXPECT().Execute(new(settingsObjectVersion)).Run(injectResponse(settingsObjectVersion{Version: "v1"})).Return(nil).Once()
			apiClient.EXPECT().GET(ctx, "/objects/"+objectID).Return(getRequest).Once()

			deleteRequest := coremock.NewRequest(t)
			deleteRequest.EXPECT().WithQueryParams(map[string]string{gen3OptimisticLockingVersionParam: "v1"}).Return(deleteRequest).Once()
			deleteRequest.EXPECT().Execute(nil).Return(nil).Once()
			apiClient.EXPECT().DELETE(ctx, "/objects/"+objectID).Return(deleteRequest).Once()

			client := NewClient(apiClient, Gen3)
			err := client.DeleteSettings(ctx, objectID)
			require.NoError(t, err)
		})

		t.Run("error fetching version", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			getRequest := coremock.NewRequest(t)
			getRequest.EXPECT().Execute(new(settingsObjectVersion)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().GET(ctx, "/objects/"+objectID).Return(getRequest).Once()

			client := NewClient(apiClient, Gen3)
			err := client.DeleteSettings(ctx, objectID)
			require.Error(t, err)
			assert.ErrorIs(t, err, errDeleteSettings)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			getRequest := coremock.NewRequest(t)
			getRequest.EXPECT().Execute(new(settingsObjectVersion)).Run(injectResponse(settingsObjectVersion{Version: "v1"})).Return(nil).Once()
			apiClient.EXPECT().GET(ctx, "/objects/"+objectID).Return(getRequest).Once()

			deleteRequest := coremock.NewRequest(t)
			deleteRequest.EXPECT().WithQueryParams(map[string]string{gen3OptimisticLockingVersionParam: "v1"}).Return(deleteRequest).Once()
			deleteRequest.EXPECT().Execute(nil).Return(errors.New("api error")).Once()
			apiClient.EXPECT().DELETE(ctx, "/objects/"+objectID).Return(deleteRequest).Once()

			client := NewClient(apiClient, Gen3)
			err := client.DeleteSettings(ctx, objectID)
			require.Error(t, err)
			assert.ErrorIs(t, err, errDeleteSettings)
		})
	})
}

func injectResponse[T any](resp T) func(any) {
	return func(arg any) {
		if target, ok := arg.(*T); ok {
			*target = resp
		}
	}
}

// matchGen2Body matches the batched array body gen2's createObject sends.
func matchGen2Body(schemaID, schemaVersion, scope string, values ...any) any {
	return mock.MatchedBy(func(arg any) bool {
		body, ok := arg.([]objectBody)
		if !ok || len(body) != len(values) {
			return false
		}

		for i, value := range values {
			if body[i].SchemaID != schemaID || body[i].SchemaVersion != schemaVersion || body[i].Scope != scope || !reflect.DeepEqual(body[i].Value, value) {
				return false
			}
		}

		return true
	})
}

// matchGen3Body matches the single-object body gen3's createObject sends.
func matchGen3Body(schemaID, schemaVersion, scope string, value any) any {
	return mock.MatchedBy(func(arg any) bool {
		body, ok := arg.(objectBody)

		return ok && body.SchemaID == schemaID && body.SchemaVersion == schemaVersion && body.Scope == scope && reflect.DeepEqual(body.Value, value)
	})
}
