package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/utils"
	"github.com/pkg/errors"
)

type monitoredEntitiesResponse struct {
	Entities   []MonitoredEntity `json:"entities"`
	TotalCount int               `json:"totalCount"`
	PageSize   int               `json:"pageSize"`
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
	Message              string
	ConstraintViolations constraintViolations `json:"constraintViolations,omitempty"`
	Code                 int
}

type constraintViolations []struct {
	ParameterLocation string
	Location          string
	Message           string
	Path              string
}

func (dtc *dynatraceClient) GetMonitoredEntitiesForKubeSystemUUID(ctx context.Context, kubeSystemUUID string) ([]MonitoredEntity, error) {
	if kubeSystemUUID == "" {
		return nil, errors.New("no kube-system namespace UUID given")
	}

	req, err := createBaseRequest(ctx, dtc.getEntitiesUrl(), http.MethodGet, dtc.apiToken, nil)
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

func (dtc *dynatraceClient) GetSettingsForMonitoredEntities(ctx context.Context, monitoredEntities []MonitoredEntity, schemaId string) (GetSettingsResponse, error) {
	if len(monitoredEntities) < 1 {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	scopes := make([]string, 0, len(monitoredEntities))
	for _, entity := range monitoredEntities {
		scopes = append(scopes, entity.EntityId)
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsUrl(true), http.MethodGet, dtc.apiToken, nil)
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

func (dtc *dynatraceClient) unmarshalToJson(res *http.Response, resDataJson any) error {
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

		return errors.New(sb.String())
	}
}
