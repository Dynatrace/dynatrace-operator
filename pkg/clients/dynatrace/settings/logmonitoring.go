// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

// GetSettingsForLogModule returns the settings response with the number of settings objects and their values.
func (c *ClientImpl) GetSettingsForLogModule(ctx context.Context, registration K8sClusterRegistration) (TotalCountSettingsResponse, error) {
	if registration.EntityScope == "" {
		return TotalCountSettingsResponse{}, nil
	}

	var resp TotalCountSettingsResponse

	err := c.api.listObjects(ctx, objectsCollection, listParams{
		SchemaID:     logMonitoringSettingsSchemaID,
		Scope:        c.api.scope(registration),
		ValidateOnly: true,
	}, &resp)
	if err != nil {
		return TotalCountSettingsResponse{}, fmt.Errorf("get logmonitoring settings: %w", err)
	}

	return resp, nil
}

// CreateLogMonitoringSetting returns the object ID of the created logmonitoring settings.
func (c *ClientImpl) CreateLogMonitoringSetting(ctx context.Context, registration K8sClusterRegistration, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error) {
	ids, err := c.api.createObject(ctx, logMonitoringSettingsSchemaID, logMonitoringSchemaVersion, c.api.scope(registration),
		logMonSettingsValue{
			SendToStorage:   true,
			Enabled:         true,
			ConfigItemTitle: clusterName,
			Matchers:        mapIngestRuleMatchers(matchers),
		})
	if err != nil {
		return "", fmt.Errorf("create logmonitoring setting: %w", err)
	}

	return singleID(ids)
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
