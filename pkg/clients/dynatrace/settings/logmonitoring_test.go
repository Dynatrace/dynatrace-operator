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

	params := map[string]string{
		validateOnlyQueryParam: "true",
		schemaIDsQueryParam:    logMonitoringSettingsSchemaID,
		scopesQueryParam:       "entity-1",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(GetSettingsResponse)).Run(injectResponse(GetSettingsResponse{TotalCount: 3})).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		resp, err := client.GetSettingsForLogModule(ctx, "entity-1")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 3}, resp)
	})

	t.Run("empty monitoredEntity", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		resp, err := client.GetSettingsForLogModule(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, GetSettingsResponse{TotalCount: 0}, resp)
	})
}

func TestCreateLogMonitoringSetting(t *testing.T) {
	ctx := t.Context()

	matchBody := func() any {
		return matchJSONBody[logMonSettingsValue](logMonitoringSettingsSchemaID, logMonitoringSchemaVersion)
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{"validateOnly": "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody()).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).Return(nil).Once()
		apiClient.EXPECT().POST(ctx, ObjectsPath).Return(request).Once()

		client := NewClient(apiClient)
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
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
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
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
		objectID, err := client.CreateLogMonitoringSetting(ctx, "scope-1", "cluster-1", nil)
		require.ErrorAs(t, err, new(notSingleEntryError))
		assert.Empty(t, objectID)
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
