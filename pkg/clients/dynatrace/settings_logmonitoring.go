package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type IngestRuleMatchers struct {
	Attribute string   `json:"attribute,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type logMonSettingsValue struct {
	ConfigItemTitle string               `json:"config-item-title"`
	Matchers        []IngestRuleMatchers `json:"matchers"`
	Enabled         bool                 `json:"enabled"`
	SendToStorage   bool                 `json:"send-to-storage"`
}

type logMonSettingsItem struct {
	ObjectID           string              `json:"objectId"`
	LogMonitoringValue logMonSettingsValue `json:"value"`
}

type posLogMonSettingsBody struct {
	SchemaId      string              `json:"schemaId"`
	SchemaVersion string              `json:"schemaVersion"`
	Scope         string              `json:"scope,omitempty"`
	Value         logMonSettingsValue `json:"value"`
}

const (
	logMonitoringSettingsSchemaId = "builtin:logmonitoring.log-storage-settings"
	schemaVersion                 = "1.0.16"
)

func (dtc *dynatraceClient) performCreateLogMonSetting(ctx context.Context, body []posLogMonSettingsBody) (string, error) { //nolint:dupl
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
		return "", errors.WithMessage(err, "error making post request to dynatrace api")
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.WithMessage(err, "error reading response")
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
		return "", errors.Errorf("response is not containing exactly one entry %s", resData)
	}

	return resDataJson[0].ObjectId, nil
}

func createBaseLogMonSettings(clusterName, schemaId string, schemaVersion string, scope string, ingestRuleMatchers []logmonitoring.IngestRuleMatchers) posLogMonSettingsBody {
	matchers := []IngestRuleMatchers{}

	for _, ingestRuleMatcher := range ingestRuleMatchers {
		matcher := IngestRuleMatchers{
			Attribute: ingestRuleMatcher.Attribute,
			Operator:  "MATCHES",
			Values:    ingestRuleMatcher.Values,
		}
		matchers = append(matchers, matcher)
	}

	base := posLogMonSettingsBody{
		SchemaId:      schemaId,
		SchemaVersion: schemaVersion,
		Value: logMonSettingsValue{
			SendToStorage:   true,
			Enabled:         true,
			ConfigItemTitle: clusterName,
			Matchers:        matchers,
		},
	}

	if scope != "" {
		base.Scope = scope
	}

	return base
}

func (dtc *dynatraceClient) CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error) {
	settings := createBaseLogMonSettings(clusterName, logMonitoringSettingsSchemaId, schemaVersion, scope, matchers)

	objectId, err := dtc.performCreateLogMonSetting(ctx, []posLogMonSettingsBody{settings})
	if err != nil {
		return "", err
	}

	return objectId, nil
}
