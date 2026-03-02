package settings

import (
	"context"
	"errors"
	"fmt"
)

const (
	kspmSettingsSchemaID      = "builtin:kubernetes.security-posture-management"
	kspmSettingsSchemaVersion = "1"
)

// GetKSPMSettings returns the settings response with the number of settings objects and their values.
func (c *Client) GetKSPMSettings(ctx context.Context, monitoredEntity string) (SettingsResponse[KSPMSettingsValue], error) {
	if monitoredEntity == "" {
		return SettingsResponse[KSPMSettingsValue]{}, nil
	}

	var resp SettingsResponse[KSPMSettingsValue]

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    kspmSettingsSchemaID,
			scopesQueryParam:       monitoredEntity,
		}).
		Execute(&resp)
	if err != nil {
		return SettingsResponse[KSPMSettingsValue]{}, fmt.Errorf("get kspm settings: %w", err)
	}

	return resp, nil
}

// CreateKSPMSetting returns the object ID of the created kspm settings.
func (c *Client) CreateKSPMSetting(ctx context.Context, monitoredEntity string, datasetPipelineEnabled bool) (string, error) {
	if monitoredEntity == "" {
		return "", errors.New("no scope (MEID) was provided for creating the KSPM setting object")
	}

	body := newPostObjectsBody(
		kspmSettingsSchemaID,
		kspmSettingsSchemaVersion,
		monitoredEntity,
		KSPMSettingsValue{
			DatasetPipelineEnabled: datasetPipelineEnabled,
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
		return "", fmt.Errorf("create kspm setting: %w", err)
	}

	return getObjectID(response)
}
