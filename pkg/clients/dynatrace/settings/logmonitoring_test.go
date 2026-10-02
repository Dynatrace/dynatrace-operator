// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSettingsForLogModule(t *testing.T) {
	ctx := t.Context()

	t.Run("empty monitoredEntity", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		resp, err := client.GetSettingsForLogModule(ctx, K8sClusterRegistration{})
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 0}, resp)
	})

	t.Run("gen2", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen2ValidateOnlyParam: "true",
			gen2SchemaIDsParam:    logMonitoringSettingsSchemaID,
			gen2ScopesParam:       "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(TotalCountSettingsResponse)).Run(injectResponse(TotalCountSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen2ObjectsPath).Return(request).Once()

		client := NewClient(apiClient, Gen2)
		resp, err := client.GetSettingsForLogModule(ctx, K8sClusterRegistration{EntityScope: "entity-1"})
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 3}, resp)
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{
			gen3SchemaIDParam: logMonitoringSettingsSchemaID,
			gen3ScopeParam:    "entity-1",
		}).Return(request).Once()
		request.EXPECT().Execute(new(TotalCountSettingsResponse)).Run(injectResponse(TotalCountSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, gen3ObjectsPath).Return(request).Once()

		client := NewClient(apiClient, Gen3)
		resp, err := client.GetSettingsForLogModule(ctx, K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "entity-1"})
		require.NoError(t, err)
		assert.Equal(t, TotalCountSettingsResponse{TotalCount: 3}, resp)
	})
}

func TestCreateLogMonitoringSetting(t *testing.T) {
	ctx := t.Context()
	value := logMonSettingsValue{
		SendToStorage:   true,
		Enabled:         true,
		ConfigItemTitle: "cluster-1",
		Matchers:        []ingestRuleMatchers{},
	}

	t.Run("gen2 uses the Monitored Entity ID as scope", func(t *testing.T) {
		registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "scope-1"}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(logMonitoringSettingsSchemaID, logMonitoringSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateLogMonitoringSetting(ctx, registration, "cluster-1", nil)
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(logMonitoringSettingsSchemaID, logMonitoringSchemaVersion, "scope-1", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectID, err := client.CreateLogMonitoringSetting(ctx, registration, "cluster-1", nil)
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
			request.EXPECT().WithJSONBody(matchGen3Body(logMonitoringSettingsSchemaID, logMonitoringSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Run(injectResponse(postObjectsResponse{ObjectID: "obj-123"})).Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateLogMonitoringSetting(ctx, registration, "cluster-1", nil)
			require.NoError(t, err)
			assert.Equal(t, "obj-123", objectID)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(logMonitoringSettingsSchemaID, logMonitoringSchemaVersion, scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectID, err := client.CreateLogMonitoringSetting(ctx, registration, "cluster-1", nil)
			require.Error(t, err)
			assert.Empty(t, objectID)
		})
	})
}

func Test_mapIngestRuleMatchers(t *testing.T) {
	tests := []struct {
		name  string
		input []logmonitoring.IngestRuleMatchers
		want  []ingestRuleMatchers
	}{
		{
			name: "empty",
			want: []ingestRuleMatchers{},
		},
		{
			name: "not empty",
			input: []logmonitoring.IngestRuleMatchers{
				{Attribute: "foo", Values: []string{"bar"}},
			},
			want: []ingestRuleMatchers{
				{Attribute: "foo", Values: []string{"bar"}, Operator: "MATCHES"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapIngestRuleMatchers(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
