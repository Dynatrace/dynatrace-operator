package settings

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRulesSetting(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{
		"validateOnly": "true",
		"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
		"scope":        "ENVIRONMENT_ID",
	}

	expectRules := []metadataenrichment.Rule{
		{Type: "type-1", Source: "source-1", Target: "target-1"},
		{Type: "type-2", Source: "source-2", Target: "target-2"},
	}

	response := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: "type-1", Source: "source-1", Target: "target-1"},
					},
				},
			},
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: "type-2", Source: "source-2", Target: "target-2"},
					},
				},
			},
		},
		TotalCount: 1,
	}

	t.Run("get rules", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(response)).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, effectiveValuesPath).Return(request)

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("no kubesystem-uuid -> error", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "", "test-entityID")
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, rules)
	})

	t.Run("no monitored-entities, use environment scope -> return not-empty, no error", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		// Should use globalScope ("environment") for scope
		request.EXPECT().WithQueryParams(map[string]string{
			"validateOnly": "true",
			"schemaIds":    "builtin:kubernetes.generic.metadata.enrichment",
			"scope":        "environment",
		}).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(response)).Return(nil).Once()
		apiClient.EXPECT().GET(ctx, effectiveValuesPath).Return(request).Once()

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "")
		require.NoError(t, err)
		assert.NotEmpty(t, rules)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("enrichment settings schema not available", func(t *testing.T) {
		apiClient := coremock.NewAPIClient(t)
		request := coremock.NewAPIRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		httpErr := &core.HTTPError{StatusCode: 404, Body: "Schema ID not found: builtin:kubernetes.generic.metadata.enrichment"}
		request.EXPECT().Execute(new(getRulesResponse)).Return(httpErr).Once()
		apiClient.EXPECT().GET(ctx, effectiveValuesPath).Return(request).Once()

		client := NewClient(apiClient)
		rules, err := client.GetRules(ctx, "kube-system-uuid", "ENVIRONMENT_ID")
		require.NoError(t, err)
		assert.Empty(t, rules)
	})
}
