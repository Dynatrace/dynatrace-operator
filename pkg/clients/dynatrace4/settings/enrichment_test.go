package settings

import (
	"context"
	"testing"

	coreMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace4/core"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetRulesSetting(t *testing.T) {
	ctx := context.Background()
	// At this point, we don't need to test the actual HTTP request (already tested in request_test), so we can mock the request and response.
	// Just test that query parameters are set correctly and the response/error is handled as expected.

	t.Run("get rules", func(t *testing.T) {
		apiClient := coreMock.NewApiClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		// Verify that the required query parameters are set correctly
		requestBuilder.On("WithContext", ctx).Return(requestBuilder)
		requestBuilder.On("WithQueryParams", map[string]string{
			"validateOnly": "true",
			"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
			"scope":        "ENVIRONMENT_ID",
		}).Return(requestBuilder)

		// Mock Execute to set the sample response
		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*GetRulesSettingsResponse); ok {
				*target = buildSampleResponse()
			}
		}).Return(nil)
		apiClient.On("GET", "/v2/settings/effectiveValues").Return(requestBuilder)

		client := NewClient(apiClient)
		rules, err := client.GetRulesSettings(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		assert.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, buildSampleResponse(), rules)
	})

	t.Run("no kubesystem-uuid -> error", func(t *testing.T) {
		apiClient := coreMock.NewApiClient(t)
		client := NewClient(apiClient)
		rules, err := client.GetRulesSettings(ctx, "", "test-entityID")
		assert.Error(t, err)
		assert.Equal(t, GetRulesSettingsResponse{}, rules)
	})

	t.Run("no monitored-entities, use environment scope -> return not-empty, no error", func(t *testing.T) {
		apiClient := coreMock.NewApiClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)
		// Should use globalScope ("environment") for scope
		requestBuilder.On("WithContext", ctx).Return(requestBuilder)
		requestBuilder.On("WithQueryParams", map[string]string{
			"validateOnly": "true",
			"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
			"scope":        "environment",
		}).Return(requestBuilder)

		requestBuilder.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			if target, ok := args[0].(*GetRulesSettingsResponse); ok {
				*target = buildSampleResponse()
			}
		}).Return(nil)
		apiClient.On("GET", "/v2/settings/effectiveValues").Return(requestBuilder)

		client := NewClient(apiClient)
		rules, err := client.GetRulesSettings(ctx, "kube-system-uuid", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, rules)
		assert.Equal(t, buildSampleResponse(), rules)
	})

	t.Run("enrichment settings schema not available", func(t *testing.T) {
		apiClient := coreMock.NewApiClient(t)
		requestBuilder := coreMock.NewRequestBuilder(t)

		// Simulate a 404 error with the schema ID in the response body
		requestBuilder.On("WithContext", ctx).Return(requestBuilder)
		requestBuilder.On("WithQueryParams", map[string]string{
			"validateOnly": "true",
			"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
			"scope":        "environment",
		}).Return(requestBuilder)

		httpErr := &core.HTTPError{
			StatusCode: 404,
			Body:       "Schema ID not found: builtin:kubernetes.generic.metadata.enrichment",
		}
		requestBuilder.On("Execute", mock.Anything).Return(httpErr)

		apiClient.On("GET", "/v2/settings/effectiveValues").Return(requestBuilder)

		client := NewClient(apiClient)
		rules, err := client.GetRulesSettings(ctx, "kube-system-uuid", "")
		assert.NoError(t, err)
		assert.Empty(t, rules)
	})
}

// buildSampleResponse returns a sample GetRulesSettingsResponse for use in tests.
func buildSampleResponse() GetRulesSettingsResponse {
	return GetRulesSettingsResponse{
		Items: []RuleItem{
			{
				Value: RulesResponseValue{
					Rules: []dynakube.EnrichmentRule{
						{Type: "type-1", Source: "source-1", Target: "target-1"},
					},
				},
			},
		},
		TotalCount: 1,
	}
}
