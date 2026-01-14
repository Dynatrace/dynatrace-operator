package settings

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
)

const (
	logMonitoringSettingsSchemaID = "builtin:logmonitoring.log-storage-settings"
	logMonitoringSchemaVersion    = "1.0.16"
)

type logMonSettingsValue struct {
	ConfigItemTitle string               `json:"config-item-title"`
	Matchers        []ingestRuleMatchers `json:"matchers"`
	Enabled         bool                 `json:"enabled"`
	SendToStorage   bool                 `json:"send-to-storage"`
}

type ingestRuleMatchers struct {
	Attribute string   `json:"attribute,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Values    []string `json:"values,omitempty"`
}

// GetSettingsForLogModule returns the settings response with the number of settings objects.
func (c *Client) GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetSettingsResponse, error) {
	if monitoredEntity == "" {
		return GetSettingsResponse{}, nil
	}

	var resp GetSettingsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    logMonitoringSettingsSchemaID,
			scopesQueryParam:       monitoredEntity,
		}).
		Execute(&resp)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("get logmonitoring settings: %w", err)
	}

	return resp, nil
}

// CreateLogMonitoringSetting returns the object ID of the created logmonitoring settings.
func (c *Client) CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error) {
	body := newPostObjectsBody(
		logMonitoringSettingsSchemaID,
		logMonitoringSchemaVersion,
		scope,
		logMonSettingsValue{
			SendToStorage:   true,
			Enabled:         true,
			ConfigItemTitle: clusterName,
			Matchers:        mapIngestRuleMatchers(matchers),
		},
	)

	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "false",
		}).
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return "", fmt.Errorf("create logmonitoring setting: %w", err)
	}

	if len(response) != 1 {
		return "", tooManyEntriesError(len(response))
	}

	return response[0].ObjectID, nil
}

func mapIngestRuleMatchers(input []logmonitoring.IngestRuleMatchers) []ingestRuleMatchers {
	output := make([]ingestRuleMatchers, len(input))
	for i, m := range input {
		output[i] = ingestRuleMatchers{
			Attribute: m.Attribute,
			Operator:  "MATCHES",
			Values:    m.Values,
		}
	}

	return output
}
