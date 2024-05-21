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
	ApiTokenHeader               = "Api-Token "
	KubernetesConnectionSchemaID = "app:dynatrace.kubernetes.connector:connection"
	KubernetesConnectionVersion  = "0.0.3"
	KubernetesConnectionName     = "edgeconnect-kubernetes-connection"
	KubernetesConnectionScope    = "environment"
)

type EnvironmentSetting struct {
	ObjectId      string                  `json:"objectId"`
	SchemaId      string                  `json:"schemaId"`
	SchemaVersion string                  `json:"schemaVersion"`
	Scope         string                  `json:"scope"`
	Value         EnvironmentSettingValue `json:"value"`
}

type EnvironmentSettingValue struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Token string `json:"token"`
}

type ObjectsList struct {
	Items       []SettingsObject
	nextPageKey string
	pageSize    int
	totalCount  int
}

type SettingsObject struct {
}

type environmentSettingsResponse struct {
	Items      []EnvironmentSetting `json:"items"`
	TotalCount int                  `json:"totalCount"`
	PageSize   int                  `json:"pageSize"`
}

// 'https://vzx38435.dev.apps.dynatracelabs.com/platform/classic/environment-api/v2/settings/objects?schemaIds=app%3Adynatrace.kubernetes.control%3Aconnection&fields=objectId%2Cvalue' \

func (c *client) GetConnectionSetting() (EnvironmentSetting, error) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, c.getSettingsObjectsUrl(), nil)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+c.ClientSecret)

	q := req.URL.Query()
	q.Add("schemaIds", KubernetesConnectionSchemaID)
	req.URL.RawQuery = q.Encode()

	response, err := c.httpClient.Do(req)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getServerResponseData(response)

	var resDataJson environmentSettingsResponse

	err = c.unmarshalToJson(response, &resDataJson)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson.Items[0], errors.WithStack(err)
}

func (c *client) CreateConnectionSetting(es EnvironmentSetting) error {
	jsonStr, err := json.Marshal(es)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, c.getSettingsObjectsUrl(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+c.ClientSecret)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getServerResponseData(response)

	return errors.WithStack(err)
}

func (c *client) UpdateConnectionSetting(es EnvironmentSetting) error {
	jsonStr, err := json.Marshal(es)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, c.getSettingsObjectsUrl(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+c.ClientSecret)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getServerResponseData(response)

	return errors.WithStack(err)
}

func (c *client) DeleteConnectionSetting(objectId string) error {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodDelete, c.getSettingsObjectsIdUrl(objectId), nil)
	if err != nil {
		return fmt.Errorf("error initializing http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", ApiTokenHeader+c.ClientSecret)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getServerResponseData(response)

	return errors.WithStack(err)
}
