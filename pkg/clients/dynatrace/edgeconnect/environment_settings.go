package edgeconnect

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

// Environment API
const (
	environmentAPIPath    = "/platform/classic/environment-api/v2"
	settingsObjectsPath   = environmentAPIPath + "/settings/objects"
	settingsObjectsIDPath = settingsObjectsPath + "/%s"
)

const (
	KubernetesConnectionSchemaID = "app:dynatrace.kubernetes.connector:connection"
	KubernetesConnectionScope    = "environment"
)

type EnvironmentSetting struct {
	ObjectID *string                 `json:"objectId"`
	SchemaID string                  `json:"schemaId"`
	Scope    string                  `json:"scope"`
	Value    EnvironmentSettingValue `json:"value"`
}

type EnvironmentSettingValue struct {
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Namespace string `json:"namespace"`
	Token     string `json:"token"`
}

type EnvironmentSettingsResponse struct {
	Items      []EnvironmentSetting `json:"items"`
	TotalCount int                  `json:"totalCount"`
	PageSize   int                  `json:"pageSize"`
}

// GetConnectionSettings get connection settings
func (c *client) GetConnectionSettings(ctx context.Context) ([]EnvironmentSetting, error) {
	qp := map[string]string{
		"schemaIds": KubernetesConnectionSchemaID,
		"scopes":    KubernetesConnectionScope,
	}

	var response EnvironmentSettingsResponse

	err := c.apiClient.GET(ctx, settingsObjectsPath).WithOAuthToken().WithQueryParams(qp).Execute(&response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get settings objects")
	}

	return response.Items, nil
}

// CreateConnectionSetting create connection setting
func (c *client) CreateConnectionSetting(ctx context.Context, es EnvironmentSetting) error {
	err := c.apiClient.POST(ctx, settingsObjectsPath).WithOAuthToken().WithJSONBody([]EnvironmentSetting{es}).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create connection setting")
	}

	return nil
}

// UpdateConnectionSetting update connection setting
func (c *client) UpdateConnectionSetting(ctx context.Context, es EnvironmentSetting) error {
	err := c.apiClient.PUT(ctx, fmt.Sprintf(settingsObjectsIDPath, *es.ObjectID)).WithOAuthToken().WithJSONBody(es).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to update connection setting")
	}

	return nil
}

// DeleteConnectionSetting deletes connection setting
func (c *client) DeleteConnectionSetting(ctx context.Context, objectID string) error {
	if objectID == "" {
		return errors.New("no connection setting object id given")
	}

	err := c.apiClient.DELETE(ctx, fmt.Sprintf(settingsObjectsIDPath, objectID)).WithOAuthToken().Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete connection setting")
	}

	return nil
}
