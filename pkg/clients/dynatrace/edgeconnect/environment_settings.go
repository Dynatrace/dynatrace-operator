package edgeconnect

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
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

// GetEnvironmentSettings get environment settings
func (c *client) GetEnvironmentSettings(ctx context.Context) ([]EnvironmentSetting, error) {
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

// CreateEnvironmentSetting create environment setting
func (c *client) CreateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error {
	err := c.apiClient.POST(ctx, settingsObjectsPath).WithOAuthToken().WithJSONBody([]EnvironmentSetting{es}).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to create environment setting")
	}

	return nil
}

// UpdateEnvironmentSetting update environment setting
func (c *client) UpdateEnvironmentSetting(ctx context.Context, es EnvironmentSetting) error {
	objectID := ptr.Deref(es.ObjectID, "")

	if objectID == "" {
		return errors.New("no environment setting object id given")
	}

	err := c.apiClient.PUT(ctx, fmt.Sprintf(settingsObjectsIDPath, objectID)).WithOAuthToken().WithJSONBody(es).Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to update environment setting")
	}

	return nil
}

// DeleteEnvironmentSetting deletes environment setting
func (c *client) DeleteEnvironmentSetting(ctx context.Context, objectID string) error {
	if objectID == "" {
		return errors.New("no environment setting object id given")
	}

	err := c.apiClient.DELETE(ctx, fmt.Sprintf(settingsObjectsIDPath, objectID)).WithOAuthToken().Execute(nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete environment setting")
	}

	return nil
}
