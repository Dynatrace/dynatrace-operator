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

type KSPMSettingsResponse struct {
	TotalCount int                `json:"totalCount"`
	Items      []KSPMSettingsItem `json:"items"`
}

type KSPMSettingsItem struct {
	ObjectID string            `json:"objectId"`
	Value    KSPMSettingsValue `json:"value"`
}

type KSPMSettingsValue struct {
	DatasetPipelineEnabled bool `json:"configurationDatasetPipelineEnabled"`
}

// GetKSPMSettings returns the settings response with the number of settings objects and their values.
func (c *client) GetKSPMSettings(ctx context.Context, monitoredEntity string) (KSPMSettingsResponse, error) {
	if monitoredEntity == "" {
		return KSPMSettingsResponse{}, nil
	}

	var resp KSPMSettingsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			schemaIDsQueryParam:    kspmSettingsSchemaID,
			scopesQueryParam:       monitoredEntity,
		}).
		Execute(&resp)
	if err != nil {
		return KSPMSettingsResponse{}, fmt.Errorf("get kspm settings: %w", err)
	}

	return resp, nil
}

// CreateKSPMSetting returns the object ID of the created kspm settings.
func (c *client) CreateKSPMSetting(ctx context.Context, monitoredEntity string, datasetPipelineEnabled bool) (string, error) {
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
