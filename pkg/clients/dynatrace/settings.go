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

type GetLogMonSettingsResponse struct {
	Items      []logMonSettingsItem `json:"items"`
	TotalCount int                  `json:"totalCount"`
}

type postSettingsResponse struct {
	ObjectId string `json:"objectId"`
}

const (
	pageSizeQueryParam = "pageSize"
	entitiesPageSize   = "500"

	entitySelectorQueryParam       = "entitySelector"
	kubernetesEntitySelectorFormat = "type(KUBERNETES_CLUSTER),kubernetesClusterId(%s)"

	fromQueryParam = "from"
	entitiesFrom   = "-365d"

	fieldsQueryParam     = "fields"
	entitiesNeededFields = "+lastSeenTms"

	schemaIDsQueryParam = "schemaIds"
	scopesQueryParam    = "scopes"
)

func (dtc *dynatraceClient) GetMonitoredEntitiesForKubeSystemUUID(ctx context.Context, kubeSystemUUID string) ([]MonitoredEntity, error) {
	if kubeSystemUUID == "" {
		return nil, errors.New("no kube-system namespace UUID given")
	}

	req, err := createBaseRequest(ctx, dtc.getEntitiesUrl(), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add(pageSizeQueryParam, entitiesPageSize)
	q.Add(entitySelectorQueryParam, fmt.Sprintf(kubernetesEntitySelectorFormat, kubeSystemUUID))
	q.Add(fromQueryParam, entitiesFrom)
	q.Add(fieldsQueryParam, entitiesNeededFields)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("check if ME exists failed")

		return nil, err
	}

	var resDataJson monitoredEntitiesResponse

	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return nil, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson.Entities, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity *MonitoredEntity, schemaId string) (GetSettingsResponse, error) {
	if monitoredEntity == nil {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsUrl(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, schemaId)
	q.Add(scopesQueryParam, monitoredEntity.EntityId)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve MEs")

		return GetSettingsResponse{}, err
	}

	var resDataJson GetSettingsResponse

	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("error parsing response body: %w", err)
	}

	return resDataJson, nil
}

func (dtc *dynatraceClient) GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetLogMonSettingsResponse, error) {
	if monitoredEntity == "" {
		return GetLogMonSettingsResponse{TotalCount: 0}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsUrl(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetLogMonSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, logMonitoringSettingsSchemaId)
	q.Add(scopesQueryParam, monitoredEntity)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve logmonitoring settings")

		return GetLogMonSettingsResponse{}, err
	}

	var resDataJson GetLogMonSettingsResponse

	err = dtc.unmarshalToJson(res, &resDataJson)
	if err != nil {
		return GetLogMonSettingsResponse{}, fmt.Errorf("error parsing response body: %w", err)
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
		var se serverErrorResponse
		if err := json.Unmarshal(response, &se); err != nil {
			return fmt.Errorf("response error: %d, can't unmarshal json response", statusCode)
		}

		return fmt.Errorf("response error: %d, %s", statusCode, se.ErrorMessage.Message)
	} else {
		var se []serverErrorResponse
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
