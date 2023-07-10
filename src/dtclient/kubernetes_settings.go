package dtclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type postKubernetesSettings struct {
	Label            string `json:"label"`
	ClusterIdEnabled bool   `json:"clusterIdEnabled"`
	ClusterId        string `json:"clusterId"`
	Enabled          bool   `json:"enabled"`
	*monitoringSettings
}

type monitoringSettings struct {
	CloudApplicationPipelineEnabled bool `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool `json:"openMetricsPipelineEnabled"`
	EventProcessingActive           bool `json:"eventProcessingActive"`
	EventProcessingV2Active         bool `json:"eventProcessingV2Active"`
	FilterEvents                    bool `json:"filterEvents"`
}

func (d *postKubernetesSettings) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &d.monitoringSettings)
}

type postKubernetesSettingsBody struct {
	SchemaId      string                 `json:"schemaId"`
	SchemaVersion string                 `json:"schemaVersion"`
	Scope         string                 `json:"scope,omitempty"`
	Value         postKubernetesSettings `json:"value"`
}

type monitoredEntitiesResponse struct {
	TotalCount int               `json:"totalCount"`
	PageSize   int               `json:"pageSize"`
	Entities   []MonitoredEntity `json:"entities"`
}

type MonitoredEntity struct {
	EntityId    string `json:"entityId"`
	DisplayName string `json:"displayName"`
	LastSeenTms int64  `json:"lastSeenTms"`
}

type GetSettingsResponse struct {
	TotalCount int `json:"totalCount"`
}

type postSettingsResponse struct {
	ObjectId string `json:"objectId"`
}

type getSchemasResponse struct {
	Version string `json:"version"`
}

type getSettingsErrorResponse struct {
	ErrorMessage getSettingsError `json:"error"`
}

type getSettingsError struct {
	Code                 int
	Message              string
	ConstraintViolations constraintViolations `json:"constraintViolations,omitempty"`
}

type constraintViolations []struct {
	ParameterLocation string
	Location          string
	Message           string
	Path              string
}

const (
	schemaId                                    = "builtin:cloud.kubernetes"
	defaultSchemaVersion                        = "1.0.27"
	hierarchicalMonitoringSettingsSchemaVersion = "3.0.1"
)

func (dtc *dynatraceClient) GetSchemasVersion(schemaId string) (string, error) {
	req, err := createBaseRequest(dtc.getSchemasUrl(schemaId), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return "", err
	}
	res, err := dtc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer CloseBodyAfterRequest(res)

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var resDataJson []getSchemasResponse
	err = json.Unmarshal(resData, &resDataJson)
	if err != nil {
		return "", err
	}
	return resDataJson[0].Version, nil
}

func (dtc *dynatraceClient) isKubernetesHierarchicalMonitoringSettings(schemaId string) (bool, error) {
	schemaVersion, err := dtc.GetSchemasVersion(schemaId)
	if err != nil {
		return false, err
	}
	schemaSemVer := strings.Split(schemaVersion, ".")
	major, err := strconv.Atoi(schemaSemVer[0])
	if err != nil {
		return false, err
	}
	return major >= 3, nil
}

func (dtc *dynatraceClient) performCreateOrUpdateKubernetesSetting(body []postKubernetesSettingsBody) (string, error) {
	bodyData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := createBaseRequest(dtc.getSettingsUrl(false), http.MethodPost, dtc.apiToken, bytes.NewReader(bodyData))
	if err != nil {
		return "", err
	}

	res, err := dtc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making post request to dynatrace api: %w", err)
	}
	defer CloseBodyAfterRequest(res)

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusCreated {
		return "", handleErrorArrayResponseFromAPI(resData, res.StatusCode)
	}

	var resDataJson []postSettingsResponse
	err = json.Unmarshal(resData, &resDataJson)
	if err != nil {
		return "", err
	}

	if len(resDataJson) != 1 {
		return "", fmt.Errorf("response is not containing exactly one entry %s", resData)
	}

	return resDataJson[0].ObjectId, nil
}

func (dtc *dynatraceClient) getKubernetesSettingBody(clusterLabel, kubeSystemUUID, scope string, schemaId string) ([]postKubernetesSettingsBody, error) {
	isK8sHierarchicalMonitoringSettings, err := dtc.isKubernetesHierarchicalMonitoringSettings(schemaId)
	if err != nil {
		log.Info("error on GetSchemasVersion", "error", err)
	}
	currentSchemaVersion := defaultSchemaVersion
	if isK8sHierarchicalMonitoringSettings {
		currentSchemaVersion = hierarchicalMonitoringSettingsSchemaVersion
	}
	body := []postKubernetesSettingsBody{
		{
			SchemaId:      schemaId,
			SchemaVersion: currentSchemaVersion,
			Value: postKubernetesSettings{
				Enabled:          true,
				Label:            clusterLabel,
				ClusterIdEnabled: true,
				ClusterId:        kubeSystemUUID,
			},
		},
	}
	if scope != "" {
		body[0].Scope = scope
	}
	if !isK8sHierarchicalMonitoringSettings {
		ms := monitoringSettings{
			CloudApplicationPipelineEnabled: true,
			OpenMetricsPipelineEnabled:      false,
			EventProcessingActive:           false,
			FilterEvents:                    false,
			EventProcessingV2Active:         false,
		}

		body[0].Value.monitoringSettings = &ms
	}
	return body, nil
}

func (dtc *dynatraceClient) CreateOrUpdateKubernetesSetting(clusterLabel, kubeSystemUUID, scope string) (string, error) {
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}
	body, err := dtc.getKubernetesSettingBody(clusterLabel, kubeSystemUUID, scope, schemaId)
	if err != nil {
		return "", err
	}

	return dtc.performCreateOrUpdateKubernetesSetting(body)
}

func (dtc *dynatraceClient) GetMonitoredEntitiesForKubeSystemUUID(kubeSystemUUID string) ([]MonitoredEntity, error) {
	if kubeSystemUUID == "" {
		return nil, errors.New("no kube-system namespace UUID given")
	}

	req, err := createBaseRequest(dtc.getEntitiesUrl(), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("pageSize", "500")
	q.Add("entitySelector", fmt.Sprintf("type(KUBERNETES_CLUSTER),kubernetesClusterId(%s)", kubeSystemUUID))
	q.Add("from", "-365d")
	q.Add("fields", "+lastSeenTms")
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)

	if err != nil {
		log.Info("check if ME exists failed")
		return nil, err
	}

	defer CloseBodyAfterRequest(res)

	var resDataJson monitoredEntitiesResponse
	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson.Entities, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntities(monitoredEntities []MonitoredEntity) (GetSettingsResponse, error) {
	if len(monitoredEntities) < 1 {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	scopes := make([]string, 0, len(monitoredEntities))
	for _, entity := range monitoredEntities {
		scopes = append(scopes, entity.EntityId)
	}

	req, err := createBaseRequest(dtc.getSettingsUrl(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add("schemaIds", "builtin:cloud.kubernetes")
	q.Add("scopes", strings.Join(scopes, ","))
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)

	if err != nil {
		log.Info("failed to retrieve MEs")
		return GetSettingsResponse{}, err
	}

	defer CloseBodyAfterRequest(res)

	var resDataJson GetSettingsResponse
	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson, nil
}

func (dtc *dynatraceClient) unmarshalToJson(res *http.Response, resDataJson interface{}) error {
	resData, err := dtc.getServerResponseData(res)

	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	err = json.Unmarshal(resData, resDataJson)

	if err != nil {
		return fmt.Errorf("error parsing response body: %w", err)
	}

	return nil
}

func handleErrorArrayResponseFromAPI(response []byte, statusCode int) error {
	if statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized {
		var se getSettingsErrorResponse
		if err := json.Unmarshal(response, &se); err != nil {
			return fmt.Errorf("response error: %d, can't unmarshal json response", statusCode)
		}
		return fmt.Errorf("response error: %d, %s", statusCode, se.ErrorMessage.Message)
	} else {
		var se []getSettingsErrorResponse
		if err := json.Unmarshal(response, &se); err != nil {
			return fmt.Errorf("response error: %d, can't unmarshal json response", statusCode)
		}

		var sb strings.Builder
		sb.WriteString("[Settings Creation]: could not create the Kubernetes setting for the following reason:\n")

		for _, errorResponse := range se {
			sb.WriteString(fmt.Sprintf("[%s; Code: %d\n", errorResponse.ErrorMessage.Message, errorResponse.ErrorMessage.Code))
			for _, constraintViolation := range errorResponse.ErrorMessage.ConstraintViolations {
				sb.WriteString(fmt.Sprintf("\t- %s\n", constraintViolation.Message))
			}
			sb.WriteString("]\n")
		}

		return fmt.Errorf(sb.String())
	}
}
