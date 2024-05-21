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
	settingsObjectsUrl := c.getSettingsObjectsUrl()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, settingsObjectsUrl, nil)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error initializing http request: %w", err)
	}

	q := req.URL.Query()
	q.Add("schemaIds", KubernetesConnectionSchemaID)
	q.Add("schemaVersion", KubernetesConnectionVersion)
	req.URL.RawQuery = q.Encode()

	response, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error making post request to dynatrace api: %w\nrequest:%v\nresponse:%v", err, req, response)
	}

	responseData, err := c.getServerResponseData(response)
	if err != nil {
		return EnvironmentSetting{}, err
	}

	var resDataJson environmentSettingsResponse

	err = json.Unmarshal(responseData, &resDataJson)
	if err != nil {
		return EnvironmentSetting{}, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson.Items[0], nil
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

	response, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getServerResponseData(response)

	return errors.WithStack(err)
}
