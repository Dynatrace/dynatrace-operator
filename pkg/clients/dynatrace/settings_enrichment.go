package dynatrace

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
)

const (
	MetadataEnrichmentSettingsSchemaId = "builtin:kubernetes.generic.metadata.enrichment"
	scopeQueryParam                    = "scope"
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

func (dtc *dynatraceClient) GetRulesSettings(ctx context.Context, kubeSystemUUID string) (GetRulesSettingsResponse, error) {
	if kubeSystemUUID == "" {
		return GetRulesSettingsResponse{}, errors.New("no kube-system namespace UUID given")
	}

	monitoredEntities, err := dtc.GetMonitoredEntitiesForKubeSystemUUID(ctx, kubeSystemUUID)
	if err != nil {
		log.Info("failed to retrieve MEs")

		return GetRulesSettingsResponse{}, err
	}

	if len(monitoredEntities) == 0 {
		log.Info("no MEs found, skipping getting enrichment rules")

		return GetRulesSettingsResponse{}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getEffectiveSettingsUrl(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetRulesSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, MetadataEnrichmentSettingsSchemaId)
	q.Add(scopeQueryParam, findLatestEntity(monitoredEntities).EntityId)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve enrichment rules")

		return GetRulesSettingsResponse{}, err
	}

	var resDataJson GetRulesSettingsResponse

	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return GetRulesSettingsResponse{}, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson, nil
}

func findLatestEntity(monitoredEntities []MonitoredEntity) MonitoredEntity {
	latest := monitoredEntities[0]
	for _, entity := range monitoredEntities {
		if entity.LastSeenTms < latest.LastSeenTms {
			latest = entity
		}
	}

	return latest
}
