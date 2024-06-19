package edgeconnect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

const (
	KubernetesConnectionSchemaID = "app:dynatrace.kubernetes.connector:connection"
	KubernetesConnectionVersion  = "0.1.5"
	KubernetesConnectionScope    = "environment"
)

type EnvironmentSetting struct {
	ObjectId      *string                 `json:"objectId"`
	SchemaId      string                  `json:"schemaId"`
	SchemaVersion string                  `json:"schemaVersion"`
	Scope         string                  `json:"scope"`
	Value         EnvironmentSettingValue `json:"value"`
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

func (c *client) GetConnectionSetting(uid string) (EnvironmentSetting, error) {
	settingsObjectsUrl := c.getSettingsObjectsUrl()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, settingsObjectsUrl, nil)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error initializing http request: %w", err)
	}

	q := req.URL.Query()
	q.Add("schemaIds", KubernetesConnectionSchemaID)
	q.Add("scopes", KubernetesConnectionScope)

	req.URL.RawQuery = q.Encode()

	response, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	responseData, err := c.getSettingsApiResponseData(response)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error getting server response data: %w", err)
	}

	var resDataJson EnvironmentSettingsResponse

	err = json.Unmarshal(responseData, &resDataJson)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error parsing response body: %w", err)
	}

	for _, item := range resDataJson.Items {
		if item.Value.UID == uid {
			return item, nil
		}
	}

	return EnvironmentSetting{}, nil
}

func (c *client) CreateConnectionSetting(es EnvironmentSetting) error {
	jsonStr, err := json.Marshal([]EnvironmentSetting{es})
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, c.getSettingsObjectsUrl(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)

	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	_, err = c.getSettingsApiResponseData(response)

	if err != nil {
		return fmt.Errorf("error reading response data: %w", err)
	}

	return nil
}

func (c *client) UpdateConnectionSetting(es EnvironmentSetting) error {
	es.SchemaVersion = KubernetesConnectionVersion

	jsonStr, err := json.Marshal(es)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, c.getSettingsObjectsIdUrl(*es.ObjectId), bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getSettingsApiResponseData(response)

	if err != nil {
		return fmt.Errorf("error reading response data: %w", err)
	}

	return nil
}

func (c *client) DeleteConnectionSetting(objectId string) error {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodDelete, c.getSettingsObjectsIdUrl(objectId), nil)
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	response, err := c.httpClient.Do(req)

	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	_, err = c.getSettingsApiResponseData(response)

	if err != nil {
		return fmt.Errorf("error reading response data: %w", err)
	}

	return nil
}
