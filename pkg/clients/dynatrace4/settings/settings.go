package settings

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
	"github.com/pkg/errors"
)

type client struct {
	apiClient core.APIClient
}

var _ Client = (*client)(nil)

// NewClient creates a new Settings API client
func NewClient(apiClient core.APIClient) Client {
	return &client{
		apiClient: apiClient,
	}
}

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

	schemaIDsQueryParam        = "schemaIds"
	kubernetesSettingsSchemaID = "builtin:cloud.kubernetes"
)

func (dtc *client) GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error) {
	if kubeSystemUUID == "" {
		return K8sClusterME{}, errors.New("no kube-system namespace UUID given")
	}

	var response getSettingsForKubeSystemUUIDResponse

	err := dtc.apiClient.GET(ObjectsPath).
		WithContext(ctx).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: trueQueryParamValue,
			pageSizeQueryParam:     entitiesPageSize,
			schemaIDsQueryParam:    kubernetesSettingsSchemaID,
			fieldsQueryParam:       kubernetesSettingsNeededFields,
			filterQueryParam:       fmt.Sprintf("value.clusterId='%s'", kubeSystemUUID),
		}).
		Execute(&response)
	if err != nil {
		log.Info("request for kubernetes setting exists failed")

		return K8sClusterME{}, err
	}

	if len(response.Settings) == 0 {
		log.Info("no kubernetes settings object according to API", "resp", response)

		return K8sClusterME{}, nil
	}

	return K8sClusterME{
		ID:   response.Settings[0].EntityID,
		Name: response.Settings[0].Value.Label,
	}, nil
}

func (dtc *client) GetK8sClusterMEDeleteThisMethod(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error) {
	if kubeSystemUUID == "" {
		return K8sClusterME{}, errors.New("no kube-system namespace UUID given")
	}

	var response getSettingsForKubeSystemUUIDResponse

	err := dtc.apiClient.GET(ObjectsPath).
		WithContext(ctx).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: trueQueryParamValue,
			pageSizeQueryParam:     entitiesPageSize,
			schemaIDsQueryParam:    "kubernetesSettingsSchemaID",
			fieldsQueryParam:       kubernetesSettingsNeededFields,
			filterQueryParam:       fmt.Sprintf("value.clusterId='%s'", kubeSystemUUID),
		}).
		Execute(&response)
	if err != nil {
		log.Info("request for kubernetes setting exists failed")

		return K8sClusterME{}, err
	}

	if len(response.Settings) == 0 {
		log.Info("no kubernetes settings object according to API", "resp", response)

		return K8sClusterME{}, nil
	}

	return K8sClusterME{
		ID:   response.Settings[0].EntityID,
		Name: response.Settings[0].Value.Label,
	}, nil
}

func (dtc *client) GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error) {
	if monitoredEntity.ID == "" {
		return GetSettingsResponse{TotalCount: 0}, nil
	}

	var response GetSettingsResponse

	err := dtc.apiClient.GET(ObjectsPath).
		WithContext(ctx).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: trueQueryParamValue,
			schemaIDsQueryParam:    schemaID,
			scopesQueryParam:       monitoredEntity.ID,
		}).
		Execute(&response)
	if err != nil {
		log.Info("failed to retrieve MEs")

		return GetSettingsResponse{}, err
	}

	return response, nil
}

func (dtc *client) GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetLogMonSettingsResponse, error) {
	if monitoredEntity == "" {
		return GetLogMonSettingsResponse{TotalCount: 0}, nil
	}

	var response GetLogMonSettingsResponse

	err := dtc.apiClient.GET(ObjectsPath).
		WithContext(ctx).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: trueQueryParamValue,
			schemaIDsQueryParam:    logMonitoringSettingsSchemaID,
			scopesQueryParam:       monitoredEntity,
		}).
		Execute(&response)
	if err != nil {
		log.Info("failed to retrieve logmonitoring settings")

		return GetLogMonSettingsResponse{}, err
	}

	return response, nil
}
