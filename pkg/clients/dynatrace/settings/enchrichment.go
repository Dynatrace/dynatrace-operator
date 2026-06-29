package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	LegacyMetadataEnrichmentSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	MetadataEnrichmentSchemaID       = "builtin:ingest.enrichment.config"
	scopeQueryParam                  = "scope"
	globalScope                      = "environment"
	effectiveValuesPath              = "/v2/settings/effectiveValues"
)

// EnrichmentRuleObject holds the objectId of a single enrichment rule settings object,
// used to identify rules for deletion.
type EnrichmentRuleObject struct {
	ObjectID string `json:"objectId"`
}

type enrichmentRulesObjectsResponse struct {
	Items []EnrichmentRuleObject `json:"items"`
}

type enrichmentObjectBody[V any] struct {
	SchemaID string `json:"schemaId"`
	Scope    string `json:"scope,omitempty"`
	Value    V      `json:"value"`
}

type enrichmentRuleValue struct {
	Type        metadataenrichment.RuleType `json:"type"`
	ValueSource string                      `json:"valueSource"`
	Target      string                      `json:"target"`
}

type legacyEnrichmentValue struct {
	Rules []metadataenrichment.Rule `json:"rules"`
}

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
	Type        metadataenrichment.RuleType `json:"type"`
	Target      string                      `json:"target"`
	ValueSource string                      `json:"valueSource"`
	Condition   string                      `json:"condition"`
}

func (c *ClientImpl) getEnrichmentRuleObjectsForSchema(ctx context.Context, schemaID, scope string) ([]EnrichmentRuleObject, error) {
	var resp enrichmentRulesObjectsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			schemaIDsQueryParam: schemaID,
			scopesQueryParam:    scope,
		}).
		Execute(&resp)
	if err != nil {
		return nil, fmt.Errorf("get enrichment rule objects (%s): %w", schemaID, err)
	}

	return resp.Items, nil
}

func (c *ClientImpl) postEnrichmentObject(ctx context.Context, body any) (string, error) {
	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, ObjectsPath).
		WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return "", err
	}

	return getObjectID(response)
}

func (c *ClientImpl) GetEnrichmentRuleObjects(ctx context.Context, scope string) ([]EnrichmentRuleObject, error) {
	if scope == "" {
		return nil, errors.New("no scope provided for getting enrichment rule objects")
	}

	return c.getEnrichmentRuleObjectsForSchema(ctx, MetadataEnrichmentSchemaID, scope)
}

func (c *ClientImpl) GetLegacyEnrichmentRuleObjects(ctx context.Context, scope string) ([]EnrichmentRuleObject, error) {
	if scope == "" {
		return nil, errors.New("no scope provided for getting legacy enrichment rule objects")
	}

	return c.getEnrichmentRuleObjectsForSchema(ctx, LegacyMetadataEnrichmentSchemaID, scope)
}

// CreateEnrichmentRule creates a settings object for the given schema and scope.
// For MetadataEnrichmentSchemaID (new schema) each rule becomes its own object; exactly one rule must be provided.
// For LegacyMetadataEnrichmentSchemaID all rules are bundled into a single object.
func (c *ClientImpl) CreateEnrichmentRule(ctx context.Context, schemaID, scope string, rules []metadataenrichment.Rule) (string, error) {
	if scope == "" {
		return "", errors.New("no scope (MEID) was provided for creating the enrichment rule")
	}

	body := buildEnrichmentBody(schemaID, scope, rules)

	objectID, err := c.postEnrichmentObject(ctx, body)
	if err != nil {
		return "", fmt.Errorf("create enrichment rule (%s): %w", schemaID, err)
	}

	return objectID, nil
}

func buildEnrichmentBody(schemaID, scope string, rules []metadataenrichment.Rule) any {
	if schemaID == MetadataEnrichmentSchemaID && len(rules) == 1 {
		return []enrichmentObjectBody[enrichmentRuleValue]{{
			SchemaID: schemaID,
			Scope:    scope,
			Value: enrichmentRuleValue{
				Type:        rules[0].Type,
				ValueSource: rules[0].Source,
				Target:      rules[0].Target,
			},
		}}
	}

	return []enrichmentObjectBody[legacyEnrichmentValue]{{
		SchemaID: schemaID,
		Scope:    scope,
		Value:    legacyEnrichmentValue{Rules: rules},
	}}
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
		schemaIDsQueryParam:    LegacyMetadataEnrichmentSchemaID,
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
		log.Info("using fallback for unavailable schema", "schemaID", LegacyMetadataEnrichmentSchemaID, "fallbackSchemaID", MetadataEnrichmentSchemaID)
	} else if useNewSchema = isNewSchemaRequested(resp); useNewSchema {
		// Users can enable the use of the schema on their tenant before migrating to Latest Dynatrace environments.
		// In this case both schemas coexist, but we should still use the new one.
		log.Info("updating data with new schema enabled on tenant", "schemaID", MetadataEnrichmentSchemaID)
	} else {
		// The legacy schema was found and the toggle is not enabled
		return getRulesFromResponse(resp), nil
	}

	// Retry the request with the new schema. For managed this will always fail, but we have no practical way of knowing which environment we're running in.
	params[schemaIDsQueryParam] = MetadataEnrichmentSchemaID
	// Clear the input so that we don't keep stale data
	resp = getRulesResponse{}

	if err := c.apiClient.GET(ctx, effectiveValuesPath).WithQueryParams(params).Execute(&resp); err != nil {
		if useNewSchema || !core.IsNotFound(err) {
			// The error is either not 404 or the user enabled the new schema explicitly. In this case a missing schema is an error.
			return nil, fmt.Errorf("get rules settings for schema %s: %w", MetadataEnrichmentSchemaID, err)
		}

		// Keep the established behavior of not failing when the legacy schema is not available
		// This covers the managed use-case.
		log.Info("enrichment settings not available on cluster, skipping getting the enrichment rules", "schemaID", LegacyMetadataEnrichmentSchemaID)

		return nil, nil
	}

	rules := getRulesFromResponse(resp)
	if useNewSchema && len(rules) == 0 {
		// Rules get auto-migrated when moving to Latest Dynatrace environments, but before then it's the users responsibility to migrate them.
		// Since the user explicitly requested the new schema, missing rules at this point are not an error.
		log.Info("requested enrichment rules, but got empty response. manual migration of rules is required", "schemaID", MetadataEnrichmentSchemaID)
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
		if cfg := item.Value.ingestEnrichmentConfig; metadataenrichment.IsSupportedType(cfg.Type) {
			if cfg.Condition != "" {
				// Skip rules with conditions for now
				continue
			}

			rule := metadataenrichment.Rule{
				Type:   cfg.Type,
				Target: cfg.Target,
				Source: cfg.ValueSource,
			}

			rules = append(rules, rule)

			continue
		}

		// Old rules always have supported types
		rules = append(rules, item.Value.Rules...)
	}

	return rules
}
