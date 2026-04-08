package edgeconnect

import (
	"context"

	"github.com/pkg/errors"
)

// Environment API
const (
	environmentAPIPath    = "/platform/classic/environment-api/v2"
	settingsObjectsPath   = environmentAPIPath + "/settings/objects"
	settingsObjectsIDPath = settingsObjectsPath + "/"
)

const (
	KubernetesConnectionSchemaID = "app:dynatrace.kubernetes.connector:connection"
	KubernetesConnectionScope    = "environment"
)

type EnvironmentSetting struct {
	ObjectID string                  `json:"objectId,omitempty"`
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

type environmentSettingsResponse struct {
	Items []EnvironmentSetting `json:"items"`
}

// ListEnvironmentSettings get environment settings
func (c *Client) ListEnvironmentSettings(ctx context.Context) ([]EnvironmentSetting, error) {
	qp := map[string]string{
		"schemaIds": KubernetesConnectionSchemaID,
		"scopes":    KubernetesConnectionScope,
	}

	var response environmentSettingsResponse

	err := c.apiClient.GET(ctx, settingsObjectsPath).WithoutToken().WithQueryParams(qp).Execute(&response)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get settings objects")
	}

	return response.Items, nil
}

// CreateEnvironmentSetting create environment setting
func (c *Client) CreateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error {
	err := c.apiClient.POST(ctx, settingsObjectsPath).WithoutToken().WithJSONBody([]EnvironmentSetting{es}).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create environment setting")
	}

	return nil
}

// UpdateEnvironmentSetting update environment setting
func (c *Client) UpdateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error {
	if es.ObjectID == "" {
		return errors.New("no environment setting object id given")
	}

	err := c.apiClient.PUT(ctx, settingsObjectsIDPath+es.ObjectID).WithoutToken().WithJSONBody(es).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to update environment setting")
	}

	return nil
}

// DeleteEnvironmentSetting deletes environment setting
func (c *Client) DeleteEnvironmentSetting(ctx context.Context, objectID string) error {
	if objectID == "" {
		return errors.New("no environment setting object id given")
	}

	err := c.apiClient.DELETE(ctx, settingsObjectsIDPath+objectID).WithoutToken().Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete environment setting")
	}

	return nil
}
