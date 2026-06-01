package settings

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestGetRulesSetting(t *testing.T) {
	ctx := t.Context()

	oldParams := map[string]string{
		"validateOnly": "true",
		"schemaIds":    legacyMetadataEnrichmentSchemaID,
		"scope":        "ENVIRONMENT_ID",
	}
	newParams := map[string]string{
		"validateOnly": "true",
		"schemaIds":    metadataEnrichmentSchemaID,
		"scope":        "ENVIRONMENT_ID",
	}

	expectRules := []metadataenrichment.Rule{
		{Type: metadataenrichment.LabelRule, Source: "source-1", Target: "target-1"},
		{Type: metadataenrichment.AnnotationRule, Source: "source-2", Target: "target-2"},
	}

	oldResponse := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "source-1", Target: "target-1"},
						{Type: metadataenrichment.AnnotationRule, Source: "source-2", Target: "target-2"},
					},
				},
			},
		},
	}

	newResponse := getRulesResponse{
		Items: []ruleItem{
			{Value: ruleItemValue{ingestEnrichmentConfig: ingestEnrichmentConfig{Type: metadataenrichment.K8sLabelRule, ValueSource: "source-1", Target: "target-1"}}},
			{Value: ruleItemValue{ingestEnrichmentConfig: ingestEnrichmentConfig{Type: metadataenrichment.K8sAnnotationRule, ValueSource: "source-2", Target: "target-2"}}},
			{Value: ruleItemValue{ingestEnrichmentConfig: ingestEnrichmentConfig{Type: metadataenrichment.CustomRule, ValueSource: "source-3", Target: "target-3"}}},
		},
	}

	setFlag := func(resp getRulesResponse) getRulesResponse {
		items := make([]ruleItem, len(resp.Items))
		copy(items, resp.Items)

		for i := range items {
			items[i].Value.UseIngestEnrichmentConfigSchema = true
		}

		return getRulesResponse{Items: items}
	}

	t.Run("get rules", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(oldParams).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(oldResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once()

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("no kubesystem-uuid -> error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		settingsClient := NewClient(apiClient)
		rules, err := settingsClient.GetRules(ctx, "", "test-entityID")
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, rules)
	})

	t.Run("non 404 error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(oldParams).Return(request).Once()
		httpErr := &core.HTTPError{StatusCode: 503}
		request.EXPECT().Execute(new(getRulesResponse)).Return(httpErr).Once()
		apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once()

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.ErrorIs(t, err, httpErr)
		assert.Empty(t, rules)
	})

	t.Run("no monitored-entities, use environment scope -> return not-empty, no error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		// Should use globalScope ("environment") for scope
		request.EXPECT().WithQueryParams(map[string]string{
			"validateOnly": "true",
			"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
			"scope":        "environment",
		}).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(oldResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once()

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "")
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("new schema enabled explicitly", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectRules := []metadataenrichment.Rule{
			{Type: metadataenrichment.K8sLabelRule, Source: "source-1", Target: "target-1"},
			{Type: metadataenrichment.K8sAnnotationRule, Source: "source-2", Target: "target-2"},
		}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(setFlag(oldResponse))).Return(nil).Once(),
			// Switch to new schema
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(newResponse)).Return(nil).Once(),
		)

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("new enrichment settings schema not available", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		httpErr := &core.HTTPError{StatusCode: 404}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(setFlag(oldResponse))).Return(nil).Once(),
			// Switch to new schema
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(httpErr).Once(),
		)

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.Error(t, err)
		assert.Empty(t, rules)
	})

	t.Run("use enrichment settings schema fallback", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectRules := []metadataenrichment.Rule{
			{Type: metadataenrichment.K8sLabelRule, Source: "source-1", Target: "target-1"},
			{Type: metadataenrichment.K8sAnnotationRule, Source: "source-2", Target: "target-2"},
		}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
			// Fallback after 404 with old schema
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(newResponse)).Return(nil).Once(),
		)

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("neither enrichment settings schema available", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
			apiClient.EXPECT().GET(anyCtx, effectiveValuesPath).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
		)

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.Empty(t, rules)
	})
}

// Set up mocked calls to verify they are executed in the input order.
func expectCallOrder(calls ...*mock.Call) {
	if len(calls) < 2 {
		return
	}

	prev := calls[0]

	for _, call := range calls[1:] {
		call.NotBefore(prev)
		prev = call
	}
}

// This is just a sanity check that the models match what's returned by the API.
func Test_enrichmentSchemaModel(t *testing.T) {
	const rawDataOld = `{"items":[{"origin":"environment","value":{"rules":[{"type":"LABEL","source":"test-cost","target":"dt.cost.product"},{"type":"ANNOTATION","source":"my.test.annotation/value","target":"dt.security_context"}]}}],"totalCount":1,"pageSize":100}`
	const rawDataOldFlag = `{"items":[{"origin":"environment","value":{"rules":[{"type":"LABEL","source":"test-cost","target":"dt.cost.product"},{"type":"ANNOTATION","source":"my.test.annotation/value","target":"dt.security_context"}],"useIngestEnrichmentConfigSchema":true}}],"totalCount":1,"pageSize":100}`
	const rawDataNew = `{"items":[{"origin":"environment","value":{"type":"K8S_NAMESPACE_LABEL","valueSource":"test-label","target":"dt.cost.product"}},{"origin":"environment","value":{"type":"K8S_NAMESPACE_ANNOTATION","valueSource":"my.test.annotation/value","target":"dt.security_context"}}],"totalCount":2,"pageSize":100}`

	expectOld := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "test-cost", Target: "dt.cost.product"},
						{Type: metadataenrichment.AnnotationRule, Source: "my.test.annotation/value", Target: "dt.security_context"},
					},
				},
			},
		},
	}

	expectOldFlag := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "test-cost", Target: "dt.cost.product"},
						{Type: metadataenrichment.AnnotationRule, Source: "my.test.annotation/value", Target: "dt.security_context"},
					},
					UseIngestEnrichmentConfigSchema: true,
				},
			},
		},
	}

	expectNew := getRulesResponse{
		Items: []ruleItem{
			{Value: ruleItemValue{ingestEnrichmentConfig: ingestEnrichmentConfig{Type: metadataenrichment.K8sLabelRule, ValueSource: "test-label", Target: "dt.cost.product"}}},
			{Value: ruleItemValue{ingestEnrichmentConfig: ingestEnrichmentConfig{Type: metadataenrichment.K8sAnnotationRule, ValueSource: "my.test.annotation/value", Target: "dt.security_context"}}},
		},
	}

	tests := []struct {
		name   string
		input  string
		expect getRulesResponse
	}{
		{"old", rawDataOld, expectOld},
		{"old with flag", rawDataOldFlag, expectOldFlag},
		{"new", rawDataNew, expectNew},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resp getRulesResponse
			require.NoError(t, json.Unmarshal([]byte(test.input), &resp))
			assert.Equal(t, test.expect, resp)
		})
	}
}
