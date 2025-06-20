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
	EntityID    string `json:"entityId"`
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
	ObjectID string `json:"objectId"`
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

	req, err := createBaseRequest(ctx, dtc.getEntitiesURL(), http.MethodGet, dtc.apiToken, nil)
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

	var resDataJSON monitoredEntitiesResponse

	err = dtc.unmarshalToJSON(res, &resDataJSON)
	if err != nil {
		return nil, errors.WithMessage(err, "error parsing response body")
	}

	return resDataJSON.Entities, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity *MonitoredEntity, schemaID string) (GetSettingsResponse, error) {
	if monitoredEntity == nil {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsURL(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, schemaID)
	q.Add(scopesQueryParam, monitoredEntity.EntityID)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve MEs")

		return GetSettingsResponse{}, err
	}

	var resDataJSON GetSettingsResponse

	err = dtc.unmarshalToJSON(res, &resDataJSON)
	if err != nil {
		return GetSettingsResponse{}, errors.WithMessage(err, "error parsing response body")
	}

	return resDataJSON, nil
}

func (dtc *dynatraceClient) GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetLogMonSettingsResponse, error) {
	if monitoredEntity == "" {
		return GetLogMonSettingsResponse{TotalCount: 0}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsURL(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetLogMonSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, logMonitoringSettingsSchemaID)
	q.Add(scopesQueryParam, monitoredEntity)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve logmonitoring settings")

		return GetLogMonSettingsResponse{}, err
	}

	var resDataJSON GetLogMonSettingsResponse

	err = dtc.unmarshalToJSON(res, &resDataJSON)
	if err != nil {
		return GetLogMonSettingsResponse{}, errors.WithMessage(err, "error parsing response body")
	}

	return resDataJSON, nil
}

func (dtc *dynatraceClient) unmarshalToJSON(res *http.Response, resDataJSON any) error {
	resData, err := dtc.getServerResponseData(res)
	if err != nil {
		return errors.WithMessage(err, "error reading response body")
	}

	err = json.Unmarshal(resData, resDataJSON)
	if err != nil {
		return errors.WithMessage(err, "error parsing response body")
	}

	return nil
}

func handleErrorArrayResponseFromAPI(response []byte, statusCode int) error {
	if statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized {
		var se serverErrorResponse
		if err := json.Unmarshal(response, &se); err != nil {
			return errors.Errorf("response error: %d, can't unmarshal json response", statusCode)
		}

		return errors.Errorf("response error: %d, %s", statusCode, se.ErrorMessage.Message)
	} else {
		var se []serverErrorResponse
		if err := json.Unmarshal(response, &se); err != nil {
			return errors.Errorf("response error: %d, can't unmarshal json response", statusCode)
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
