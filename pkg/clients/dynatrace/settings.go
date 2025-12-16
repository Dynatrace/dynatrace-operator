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

type getSettingsForKubeSystemUUIDResponse struct {
	Settings   []kubernetesSetting `json:"items"`
	TotalCount int                 `json:"totalCount"`
	PageSize   int                 `json:"pageSize"`
}

type kubernetesSetting struct {
	EntityID string                 `json:"scope"`
	Value    kubernetesSettingValue `json:"value"`
}

type kubernetesSettingValue struct {
	Label string `json:"label"`
}

// K8sClusterME is representing the relevant info for a Kubernetes Cluster Monitored Entity
type K8sClusterME struct {
	ID   string
	Name string
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

	scopesQueryParam               = "scopes"
	filterQueryParam               = "filter"
	fieldsQueryParam               = "fields"
	kubernetesSettingsNeededFields = "value,scope"

	schemaIDsQueryParam = "schemaIds"
)

func (dtc *dynatraceClient) GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error) {
	if kubeSystemUUID == "" {
		return K8sClusterME{}, errors.New("no kube-system namespace UUID given")
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsURL(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return K8sClusterME{}, err
	}

	q := req.URL.Query()
	q.Add(pageSizeQueryParam, entitiesPageSize)
	q.Add(schemaIDsQueryParam, KubernetesSettingsSchemaID)
	q.Add(fieldsQueryParam, kubernetesSettingsNeededFields)
	q.Add(filterQueryParam, fmt.Sprintf("value.clusterId='%s'", kubeSystemUUID))
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("request for kubernetes setting exists failed")

		return K8sClusterME{}, err
	}

	var resDataJSON getSettingsForKubeSystemUUIDResponse

	err = dtc.unmarshalToJSON(res, &resDataJSON)
	if err != nil {
		return K8sClusterME{}, err
	}

	if len(resDataJSON.Settings) == 0 {
		log.Info("no kubernetes settings object according to API", "resp", resDataJSON)

		return K8sClusterME{}, nil
	}

	return K8sClusterME{
		ID:   resDataJSON.Settings[0].EntityID,
		Name: resDataJSON.Settings[0].Value.Label,
	}, nil
}

func (dtc *dynatraceClient) GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error) {
	if monitoredEntity.ID == "" {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsURL(true), http.MethodGet, dtc.apiToken, nil)
	if err != nil {
		return GetSettingsResponse{}, err
	}

	q := req.URL.Query()
	q.Add(schemaIDsQueryParam, schemaID)
	q.Add(scopesQueryParam, monitoredEntity.ID)
	req.URL.RawQuery = q.Encode()

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		log.Info("failed to retrieve MEs")

		return GetSettingsResponse{}, err
	}

	var resDataJSON GetSettingsResponse

	if err := dtc.unmarshalToJSON(res, &resDataJSON); err != nil {
		return GetSettingsResponse{}, err
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
		return GetLogMonSettingsResponse{}, err
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
