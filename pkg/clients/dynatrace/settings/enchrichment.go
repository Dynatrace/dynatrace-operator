package settings

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	metadataEnrichmentSettingsSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	scopeQueryParam                    = "scope"
	globalScope                        = "environment"
	effectiveValuesPath                = "/v2/settings/effectiveValues"
)

type getRulesResponse struct {
	Items      []ruleItem `json:"items"`
	TotalCount int        `json:"totalCount"`
}

type ruleItem struct {
	Value ruleItemValue `json:"value"`
}

type ruleItemValue struct {
	Rules []metadataenrichment.Rule `json:"rules"`
}

// GetRules returns metadata enrichment rules with the number of settings objects.
func (c *Client) GetRules(ctx context.Context, kubeSystemUUID, entityID string) ([]metadataenrichment.Rule, error) {
	if kubeSystemUUID == "" {
		return nil, errMissingKubeSystemUUID
	}

	scope := entityID
	if scope == "" {
		log.Info("No Monitored Entity ID, getting environment enrichment rules")

		scope = globalScope
	}

	var resp getRulesResponse

	err := c.apiClient.GET(ctx, effectiveValuesPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    metadataEnrichmentSettingsSchemaID,
			scopeQueryParam:        scope,
		}).
		Execute(&resp)
	if err != nil {
		if core.IsNotFound(err) {
			log.Info("enrichment settings not available on cluster, skipping getting the enrichment rules", "schemaID", metadataEnrichmentSettingsSchemaID)

			return nil, nil
		}

		log.Info("failed to retrieve enrichment rules")

		return nil, fmt.Errorf("get rules settings: %w", err)
	}

	var rules []metadataenrichment.Rule

	for _, item := range resp.Items {
		rules = append(rules, item.Value.Rules...)
	}

	return rules, nil
}
