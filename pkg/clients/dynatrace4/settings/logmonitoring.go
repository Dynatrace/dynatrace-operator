package settings

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
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
	SchemaID      string              `json:"schemaId"`
	SchemaVersion string              `json:"schemaVersion"`
	Scope         string              `json:"scope,omitempty"`
	Value         logMonSettingsValue `json:"value"`
}

const (
	logMonitoringSettingsSchemaID = "builtin:logmonitoring.log-storage-settings"
	schemaVersion                 = "1.0.16"
)

func (dtc *client) CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error) {
	settings := createBaseLogMonSettings(clusterName, logMonitoringSettingsSchemaID, schemaVersion, scope, matchers)

	objectID, err := dtc.performCreateLogMonSetting(ctx, []posLogMonSettingsBody{settings})
	if err != nil {
		return "", err
	}

	return objectID, nil
}

func (dtc *client) performCreateLogMonSetting(ctx context.Context, body []posLogMonSettingsBody) (string, error) {
	var response []postSettingsResponse

	err := dtc.apiClient.POST(ObjectsPath).
		WithContext(ctx).
		WithQueryParam(validateOnlyQueryParam, "false").
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return "", errors.WithMessage(err, "error making post request to dynatrace api")
	}

	if len(response) != 1 {
		return "", errors.Errorf("response is not containing exactly one entry, got %d entries", len(response))
	}

	return response[0].ObjectID, nil
}

func createBaseLogMonSettings(clusterName, schemaID, schemaVersion, scope string, ingestRuleMatchers []logmonitoring.IngestRuleMatchers) posLogMonSettingsBody {
	matchers := make([]IngestRuleMatchers, len(ingestRuleMatchers))
	for i, m := range ingestRuleMatchers {
		matchers[i] = IngestRuleMatchers{
			Attribute: m.Attribute,
			Operator:  "MATCHES",
			Values:    m.Values,
		}
	}

	return posLogMonSettingsBody{
		SchemaID:      schemaID,
		SchemaVersion: schemaVersion,
		Scope:         scope,
		Value: logMonSettingsValue{
			SendToStorage:   true,
			Enabled:         true,
			ConfigItemTitle: clusterName,
			Matchers:        matchers,
		},
	}
}
