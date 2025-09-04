package settings

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
	"github.com/pkg/errors"
)

const (
	MetadataEnrichmentSettingsSchemaID = "builtin:kubernetes.generic.metadata.enrichment"
	scopeQueryParam                    = "scope"
	globalScope                        = "environment"
)

type GetRulesSettingsResponse struct {
	Items      []RuleItem `json:"items"`
	TotalCount int        `json:"totalCount"`
}

type RuleItem struct {
	Value RulesResponseValue `json:"value"`
}

type RulesResponseValue struct {
	Rules []dynakube.EnrichmentRule `json:"rules"`
}

func (dtc *client) GetRulesSettings(ctx context.Context, kubeSystemUUID string, entityID string) (GetRulesSettingsResponse, error) {
	if kubeSystemUUID == "" {
		return GetRulesSettingsResponse{}, errors.New("no kube-system namespace UUID given")
	}

	scope := entityID
	if scope == "" {
		// if monitored entities were empty we then fallback to the enrichment-rules defined globally
		log.Info("No Monitored Entity ID, getting environment enrichment rules")

		scope = globalScope
	}

	var response GetRulesSettingsResponse

	err := dtc.apiClient.GET(EffectiveValuesPath).
		WithContext(ctx).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: trueQueryParamValue,
			schemaIDsQueryParam:    MetadataEnrichmentSettingsSchemaID,
			scopeQueryParam:        scope,
		}).
		Execute(&response)
	if err != nil {
		// Check for specific 404 error with schema ID
		httpErr := &core.HTTPError{}
		if errors.As(err, &httpErr) {
			log.Info("enrichment settings schema not available on cluster, skipping getting the enrichment rules")

			return GetRulesSettingsResponse{}, nil
		}

		log.Info("failed to retrieve enrichment rules")

		return GetRulesSettingsResponse{}, errors.WithMessage(err, "error parsing response body")
	}

	return response, nil
}
