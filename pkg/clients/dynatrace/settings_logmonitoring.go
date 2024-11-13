package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
)

type IngestRuleMatchers struct {
	Attribute string   `json:"attribute,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type Value struct {
	Enabled  bool                 `json:"enabled,omitempty"`
	Title    string               `json:"config-item-title,omitempty"`
	Storage  bool                 `json:"send-to-storage,omitempty"`
	Matchers []IngestRuleMatchers `json:"matchers,omitempty"`
}

type Item struct {
	ObjectID string `json:"objectId,omitempty"`
	Val      Value  `json:"value,omitempty"`
}

type posLogMonSettingsBody struct {
	Value         Value  `json:"value"`
	SchemaId      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion"`
	Scope         string `json:"scope,omitempty"`
}

const (
	LogMonitoringSettingsSchemaId = "builtin:logmonitoring.log-storage-settings"
)

func (dtc *dynatraceClient) performCreateLogMonSetting(ctx context.Context, body []posLogMonSettingsBody) (string, error) {
	bodyData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsUrl(false), http.MethodPost, dtc.apiToken, bytes.NewReader(bodyData))
	if err != nil {
		return "", err
	}

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		return "", fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusCreated {
		return "", handleErrorArrayResponseFromAPI(resData, res.StatusCode)
	}

	var resDataJson []postSettingsResponse

	err = json.Unmarshal(resData, &resDataJson)
	if err != nil {
		return "", err
	}

	if len(resDataJson) != 1 {
		return "", fmt.Errorf("response is not containing exactly one entry %s", resData)
	}

	return resDataJson[0].ObjectId, nil
}

func createBaseLogMonSettings(clusterName, schemaId string, schemaVersion string, scope string, ingestRuleMatchers []logmonitoring.IngestRuleMatchers) posLogMonSettingsBody {
	matchers := []IngestRuleMatchers{}
	if len(ingestRuleMatchers) > 0 {
		for _, ingestRuleMatcher := range ingestRuleMatchers {
			matcher := IngestRuleMatchers{
				Attribute: ingestRuleMatcher.Attribute,
				Operator:  "MATCHES",
				Values:    ingestRuleMatcher.Values,
			}
			matchers = append(matchers, matcher)
		}
	}

	base := posLogMonSettingsBody{
		SchemaId:      schemaId,
		SchemaVersion: schemaVersion,
		Value: Value{
			Storage:  true,
			Enabled:  true,
			Title:    clusterName,
			Matchers: matchers,
		},
	}

	if scope != "" {
		base.Scope = scope
	}

	return base
}

func (dtc *dynatraceClient) CreateLogMonitoringSetting(ctx context.Context, schemaID, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error) {
	settings := createBaseLogMonSettings(clusterName, schemaID, "1.0.0", scope, matchers)

	objectId, err := dtc.performCreateLogMonSetting(ctx, []posLogMonSettingsBody{settings})
	if err != nil {
		return "", err
	}

	return objectId, nil
}
