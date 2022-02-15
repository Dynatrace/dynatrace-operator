package dtclient

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type postKubernetesSettings struct {
	Label                           string `json:"label"`
	ClusterIdEnabled                bool   `json:"clusterIdEnabled"`
	ClusterId                       string `json:"clusterId"`
	CloudApplicationPipelineEnabled bool   `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool   `json:"openMetricsPipelineEnabled"`
	Enabled                         bool   `json:"enabled"`
	EventProcessingActive           bool   `json:"eventProcessingActive"`
	EventProcessingV2Active         bool   `json:"eventProcessingV2Active"`
	FilterEvents                    bool   `json:"filterEvents"`
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

func (dtc *dynatraceClient) CreateOrUpdateKubernetesSetting(name, kubeSystemUUID, scope string) (string, error) {
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	body := []postKubernetesSettingsBody{
		{
			SchemaId:      "builtin:cloud.kubernetes",
			SchemaVersion: "1.0.27",
			Value: postKubernetesSettings{
				Enabled:                         true,
				Label:                           name,
				ClusterIdEnabled:                true,
				ClusterId:                       kubeSystemUUID,
				CloudApplicationPipelineEnabled: true,
				OpenMetricsPipelineEnabled:      false,
				EventProcessingActive:           false,
				FilterEvents:                    false,
				EventProcessingV2Active:         false,
			},
		},
	}

	if scope == "" {
		var meID = generateKubernetesMEIdentifier(kubeSystemUUID)
		if meID != "" {
			body[0].Scope = meID
		}
	} else {
		body[0].Scope = scope
	}

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
		return "", fmt.Errorf("error making post request to dynatrace api: %s", err.Error())
	}

	resData, err := ioutil.ReadAll(res.Body)
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

	var resDataJson monitoredEntitiesResponse
	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %s", err.Error())
	}

	return resDataJson.Entities, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntities(monitoredEntities []MonitoredEntity) (GetSettingsResponse, error) {
	if len(monitoredEntities) < 1 {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	var scopes []string
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

	var resDataJson GetSettingsResponse
	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("error parsing response body: %s", err.Error())
	}

	return resDataJson, nil
}

func (dtc *dynatraceClient) unmarshalToJson(res *http.Response, resDataJson interface{}) error {
	resData, err := dtc.getServerResponseData(res)

	if err != nil {
		return fmt.Errorf("error reading response body: %s", err.Error())
	}
	err = json.Unmarshal(resData, resDataJson)

	if err != nil {
		return fmt.Errorf("error parsing response body: %s", err.Error())
	}

	return nil
}

func createBaseRequest(url, method, apiToken string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error initializing http request: %s", err.Error())
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", apiToken))

	if method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
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

func generateKubernetesMEIdentifier(kubeSystemUUID string) string {
	var hasher = fnv.New64()
	_, err := hasher.Write([]byte(kubeSystemUUID))
	if err != nil {
		return ""
	}
	var hash = hasher.Sum64()
	byteHash := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteHash, hash)
	return "KUBERNETES_CLUSTER-" + strings.ToUpper(hex.EncodeToString(byteHash))
}
