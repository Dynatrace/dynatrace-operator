package dynatrace

import (
	"context"
	"net/http"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
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
	Rules []metadataenrichment.Rule `json:"rules"`
}

func (dtc *dynatraceClient) GetRulesSettings(ctx context.Context, kubeSystemUUID string, entityID string) (GetRulesSettingsResponse, error) {
	if kubeSystemUUID == "" {
		return GetRulesSettingsResponse{}, errors.New("no kube-system namespace UUID given")
	}

	scope := entityID
	if scope == "" {
		// if monitored entities were empty we then fallback to the enrichment-rules defined globally
		log.Info("No Monitored Entity ID, getting environment enrichment rules")

		scope = globalScope
	}

	req, err := createBaseRequest(ctx, dtc.getEffectiveSettingsURL(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetRulesSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, MetadataEnrichmentSettingsSchemaID)
	q.Add(scopeQueryParam, scope)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve enrichment rules")

		return GetRulesSettingsResponse{}, err
	}

	var resDataJSON GetRulesSettingsResponse

	err = dtc.unmarshalToJSON(res, &resDataJSON)
	if err != nil {
		if strings.Contains(err.Error(), "404") && strings.Contains(err.Error(), MetadataEnrichmentSettingsSchemaID) {
			log.Info("enrichment settings schema not available on cluster, skipping getting the enrichment rules")

			return GetRulesSettingsResponse{}, nil
		}

		return GetRulesSettingsResponse{}, errors.WithMessage(err, "error parsing response body")
	}

	return resDataJSON, nil
}
