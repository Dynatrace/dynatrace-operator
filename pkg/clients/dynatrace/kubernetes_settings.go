package dynatrace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type postKubernetesSettings struct {
	Label            string `json:"label"`
	ClusterIdEnabled bool   `json:"clusterIdEnabled"`
	ClusterId        string `json:"clusterId"`
	Enabled          bool   `json:"enabled"`
	*MonitoringSettings
}

type MonitoringSettings struct {
	CloudApplicationPipelineEnabled bool `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool `json:"openMetricsPipelineEnabled"`
	EventProcessingActive           bool `json:"eventProcessingActive"`
	EventProcessingV2Active         bool `json:"eventProcessingV2Active"`
	FilterEvents                    bool `json:"filterEvents"`
}

type postKubernetesSettingsBody struct {
	SchemaId      string      `json:"schemaId"`
	SchemaVersion string      `json:"schemaVersion"`
	Scope         string      `json:"scope,omitempty"`
	Value         interface{} `json:"value"`
}

type postKubernetesAppSettings struct {
	KubernetesAppOptions kubernetesAppOptionsSettings `json:"kubernetesAppOptions"`
}
type kubernetesAppOptionsSettings struct {
	EnableKubernetesApp bool `json:"enableKubernetesApp"`
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
	SettingsSchemaId                            = "builtin:cloud.kubernetes"
	AppTransitionSchemaId                       = "builtin:app-transition.kubernetes"
	schemaVersionV1                             = "1.0.27"
	hierarchicalMonitoringSettingsSchemaVersion = "3.0.0"
	appTransitionSchemaVersion                  = "1.0.1"
)

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
	defer utils.CloseBodyAfterRequest(res)

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
func createPostKubernetesSettings(clusterLabel, kubeSystemUUID string) postKubernetesSettings {
	settings := postKubernetesSettings{
		Enabled:          true,
		Label:            clusterLabel,
		ClusterIdEnabled: true,
		ClusterId:        kubeSystemUUID,
	}
	return settings
}

func createBaseKubernetesSettings(postK8sSettings any, schemaId string, schemaVersion string, scope string) postKubernetesSettingsBody {
	base := postKubernetesSettingsBody{
		SchemaId:      schemaId,
		SchemaVersion: schemaVersion,
		Value:         postK8sSettings,
	}
	if scope != "" {
		base.Scope = scope
	}
	return base
}

func createV1KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope string) []postKubernetesSettingsBody {
	postK8sSettings := createPostKubernetesSettings(clusterLabel, kubeSystemUUID)
	ms := MonitoringSettings{
		CloudApplicationPipelineEnabled: true,
		OpenMetricsPipelineEnabled:      false,
		EventProcessingActive:           false,
		FilterEvents:                    false,
		EventProcessingV2Active:         false,
	}
	postK8sSettings.MonitoringSettings = &ms

	settings := createBaseKubernetesSettings(postK8sSettings, SettingsSchemaId, schemaVersionV1, scope)

	return []postKubernetesSettingsBody{settings}
}

func createV3KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope string) []postKubernetesSettingsBody {
	settings := createBaseKubernetesSettings(
		createPostKubernetesSettings(clusterLabel, kubeSystemUUID),
		SettingsSchemaId,
		hierarchicalMonitoringSettingsSchemaVersion,
		scope)
	settings.SchemaVersion = hierarchicalMonitoringSettingsSchemaVersion
	return []postKubernetesSettingsBody{settings}
}

func (dtc *dynatraceClient) CreateOrUpdateKubernetesSetting(clusterLabel, kubeSystemUUID, scope string) (string, error) {
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}
	body := createV3KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope)
	objectId, err := dtc.performCreateOrUpdateKubernetesSetting(body)
	if err != nil {
		if strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
			body = createV1KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope)
			return dtc.performCreateOrUpdateKubernetesSetting(body)
		} else {
			return "", err
		}
	}
	return objectId, nil
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

	defer utils.CloseBodyAfterRequest(res)

	var resDataJson monitoredEntitiesResponse
	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson.Entities, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntities(monitoredEntities []MonitoredEntity, schemaId string) (GetSettingsResponse, error) {
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
	q.Add("schemaIds", schemaId)
	q.Add("scopes", strings.Join(scopes, ","))
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)

	if err != nil {
		log.Info("failed to retrieve MEs")
		return GetSettingsResponse{}, err
	}

	defer utils.CloseBodyAfterRequest(res)

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

func (dtc *dynatraceClient) CreateOrUpdateKubernetesAppSetting(scope string) (string, error) {
	settings := createBaseKubernetesSettings(postKubernetesAppSettings{
		kubernetesAppOptionsSettings{
			EnableKubernetesApp: true,
		},
	}, AppTransitionSchemaId, appTransitionSchemaVersion, scope)
	objectId, err := dtc.performCreateOrUpdateKubernetesSetting([]postKubernetesSettingsBody{settings})
	if err != nil {
		return "", err
	}

	return objectId, nil
}
