package settings

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	legacyMetadataEnrichmentSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	metadataEnrichmentSchemaID       = "builtin:ingest.enrichment.config"
	scopeQueryParam                  = "scope"
	globalScope                      = "environment"
	effectiveValuesPath              = "/v2/settings/effectiveValues"
)

type getRulesResponse struct {
	Items []ruleItem `json:"items"`
}

type ruleItem struct {
	Value ruleItemValue `json:"value"`
}

type ruleItemValue struct {
	Rules []metadataenrichment.Rule `json:"rules"`

	// If this flag is enabled, the client should retry using the new schema
	UseIngestEnrichmentConfigSchema bool `json:"useIngestEnrichmentConfigSchema"`

	// These fields are embedded into the value instead of part of a rules list, when using the builtin:ingest.enrichment.config schema.
	// Group them in a struct for easy emptiness check.
	ingestEnrichmentConfig
}

type ingestEnrichmentConfig struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	ValueSource string `json:"valueSource"`
}

// Check whether necessary fields are set. For now, keep it simple and just check whether any field is set.
func (c ingestEnrichmentConfig) isEmpty() bool {
	return c == ingestEnrichmentConfig{}
}

// GetRules returns metadata enrichment rules.
func (c *ClientImpl) GetRules(ctx context.Context, kubeSystemUUID, entityID string) ([]metadataenrichment.Rule, error) {
	ctx, log := logd.NewFromContext(ctx, "dtclient-settings")

	if kubeSystemUUID == "" {
		return nil, errMissingKubeSystemUUID
	}

	scope := entityID
	if scope == "" {
		log.Info("No Monitored Entity ID, getting environment enrichment rules")

		scope = globalScope
	}

	params := map[string]string{
		validateOnlyQueryParam: "true",
		schemaIDsQueryParam:    legacyMetadataEnrichmentSchemaID,
		scopeQueryParam:        scope,
	}

	var (
		resp         getRulesResponse
		useNewSchema bool
	)

	if err := c.apiClient.GET(ctx, effectiveValuesPath).WithQueryParams(params).Execute(&resp); err != nil {
		if !core.IsNotFound(err) {
			return nil, err
		}

		// The schema is expected to be missing in two cases, the tenant is too new or too old.
		log.Info("using fallback for unavailable schema", "schemaID", legacyMetadataEnrichmentSchemaID, "fallbackSchemaID", metadataEnrichmentSchemaID)
	} else if useNewSchema = isNewSchemaRequested(resp); useNewSchema {
		// Users can enable the use of the schema on their tenant before migrating to Latest Dynatrace environments.
		// In this case both schemas coexist, but we should still use the new one.
		log.Info("updating data with new schema enabled on tenant", "schemaID", metadataEnrichmentSchemaID)
	} else {
		// The legacy schema was found and the toggle is not enabled
		return getRulesFromResponse(resp), nil
	}

	// Retry the request with the new schema. For managed this will always fail, but we have no practical way of knowing which environment we're running in.
	params[schemaIDsQueryParam] = metadataEnrichmentSchemaID
	// Clear the input so that we don't keep stale data
	resp = getRulesResponse{}

	if err := c.apiClient.GET(ctx, effectiveValuesPath).WithQueryParams(params).Execute(&resp); err != nil {
		if !useNewSchema && core.IsNotFound(err) {
			// Keep the established behavior of not failing when the legacy schema is not available
			// This covers the managed use-case.
			log.Info("enrichment settings not available on cluster, skipping getting the enrichment rules", "schemaID", legacyMetadataEnrichmentSchemaID)

			return nil, nil
		}

		// The error is either not 404 or the user enabled the new schema explicitly. In this case a missing schema is an error.
		return nil, fmt.Errorf("get rules settings for schema %s: %w", metadataEnrichmentSchemaID, err)
	}

	rules := getRulesFromResponse(resp)
	if useNewSchema && len(rules) == 0 {
		// Rules get auto-migrated when moving to Latest Dynatrace environments, but before then it's the users responsibility to migrate them.
		// Since the user explicitly requested the new schema, missing rules at this point are not an error.
		log.Info("requested enrichment rules, but got empty response. manual migration of rules is required", "schemaID", metadataEnrichmentSchemaID)
	}

	return rules, nil
}

func isNewSchemaRequested(resp getRulesResponse) bool {
	for _, item := range resp.Items {
		if item.Value.UseIngestEnrichmentConfigSchema {
			return true
		}
	}

	return false
}

func getRulesFromResponse(resp getRulesResponse) []metadataenrichment.Rule {
	var rules []metadataenrichment.Rule

	// In practice, this loop is only actually required for the new schema where each rule is a separate item.
	// The legacy schema put all rules into a single item's value.
	for _, item := range resp.Items {
		if cfg := item.Value.ingestEnrichmentConfig; !cfg.isEmpty() {
			rule := metadataenrichment.Rule{
				Type:   metadataenrichment.RuleType(cfg.Type),
				Target: cfg.Target,
				Source: cfg.ValueSource,
			}

			rules = append(rules, rule)
		} else {
			rules = append(rules, item.Value.Rules...)
		}
	}

	return rules
}
