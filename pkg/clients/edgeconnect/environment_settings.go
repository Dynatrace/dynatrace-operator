package edgeconnect

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
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

func (c *client) GetConnectionSettings() ([]EnvironmentSetting, error) {
	settingsObjectsURL := c.getSettingsObjectsURL()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, settingsObjectsURL, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "error initializing http request")
	}

	q := req.URL.Query()
	q.Add("schemaIds", KubernetesConnectionSchemaID)
	q.Add("scopes", KubernetesConnectionScope)

	req.URL.RawQuery = q.Encode()

	response, err := c.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return nil, errors.WithMessage(err, "error making post request to dynatrace api")
	}

	responseData, err := c.getSettingsAPIResponseData(response)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting server response data")
	}

	var resDataJSON EnvironmentSettingsResponse

	err = json.Unmarshal(responseData, &resDataJSON)
	if err != nil {
		return nil, errors.WithMessage(err, "error parsing response body")
	}

	return resDataJSON.Items, nil
}

func (c *client) CreateConnectionSetting(es EnvironmentSetting) error {
	jsonStr, err := json.Marshal([]EnvironmentSetting{es})
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, c.getSettingsObjectsURL(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return errors.WithMessage(err, "error initializing http request")
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)

	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return errors.WithMessage(err, "error making post request to dynatrace api")
	}

	_, err = c.getSettingsAPIResponseData(response)
	if err != nil {
		return errors.WithMessage(err, "error reading response data")
	}

	return nil
}

func (c *client) UpdateConnectionSetting(es EnvironmentSetting) error {
	jsonStr, err := json.Marshal(es)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, c.getSettingsObjectsIDURL(*es.ObjectID), bytes.NewBuffer(jsonStr))
	if err != nil {
		return errors.WithMessage(err, "error initializing http request")
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "error making post request to dynatrace api")
	}

	defer utils.CloseBodyAfterRequest(response)

	_, err = c.getSettingsAPIResponseData(response)
	if err != nil {
		return errors.WithMessage(err, "error reading response data")
	}

	return nil
}

func (c *client) DeleteConnectionSetting(objectID string) error {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodDelete, c.getSettingsObjectsIDURL(objectID), nil)
	if err != nil {
		return errors.WithMessage(err, "error initializing http request")
	}

	response, err := c.httpClient.Do(req)

	defer utils.CloseBodyAfterRequest(response)

	if err != nil {
		return errors.WithMessage(err, "error making post request to dynatrace api")
	}

	_, err = c.getSettingsAPIResponseData(response)
	if err != nil {
		return errors.WithMessage(err, "error reading response data")
	}

	return nil
}
