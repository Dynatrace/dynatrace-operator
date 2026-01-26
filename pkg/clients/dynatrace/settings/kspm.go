package settings

import (
	"context"
	"errors"
	"fmt"
)

const (
	kspmSettingsSchemaID      = "builtin:kubernetes.security-posture-management"
	kspmSettingsSchemaVersion = ""
)

type kspmSettingsValue struct {
	DatasetPipelineEnabled bool `json:"configurationDatasetPipelineEnabled"`
}

// GetSettingsForLogModule returns the settings response with the number of settings objects.
func (c *Client) GetKSPMSettings(ctx context.Context, monitoredEntity string) (GetSettingsResponse, error) {
	if monitoredEntity == "" {
		return GetSettingsResponse{}, nil
	}

	var resp GetSettingsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    kspmSettingsSchemaID,
			scopesQueryParam:       monitoredEntity,
		}).
		Execute(&resp)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("get kspm settings: %w", err)
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
		kspmSettingsValue{
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

	if len(response) != 1 {
		return "", tooManyEntriesError(len(response))
	}

	return response[0].ObjectID, nil
}
