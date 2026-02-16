// Package settings implements a client for the v2 settings API.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/go-logr/logr"
)

var log = logd.Get().WithName("dtclient-settings")

const (
	validateOnlyQueryParam = "validateOnly"
	pageSizeQueryParam     = "pageSize"
	entitiesPageSize       = "500"

	scopesQueryParam               = "scopes"
	filterQueryParam               = "filter"
	fieldsQueryParam               = "fields"
	kubernetesSettingsNeededFields = "value,scope"

	schemaIDsQueryParam        = "schemaIds"
	kubernetesSettingsSchemaID = "builtin:cloud.kubernetes"

	ObjectsPath = "/v2/settings/objects"
)

var errMissingKubeSystemUUID = errors.New("no kube-system namespace UUID given")

type APIClient interface {
	// GetK8sClusterME returns the Kubernetes Cluster Monitored Entity for the give kubernetes cluster.
	// Uses the `settings.read` scope to list the `builtin:cloud.kubernetes` settings.
	//   - Only 1 such setting exists per tenant per kubernetes cluster
	//   - The `scope` for the setting is the ID (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) of the Kubernetes Cluster Monitored Entity
	//   - The `label` of the setting is the Name (example: my-dynakube) of the Kubernetes Cluster Monitored Entity
	//
	// In case 0 settings are found, so no Kubernetes Cluster Monitored Entity exists, we return an empty object, without an error.
	GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error)
	// GetSettingsForMonitoredEntity returns the settings response with the number of settings objects.
	GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error)
	// GetSettingsForLogModule returns the settings response with the number of settings objects.
	GetSettingsForLogModule(ctx context.Context, monitoredEntity string) (GetSettingsResponse, error)
	// GetRules returns metadata enrichment rules with the number of settings objects.
	GetRules(ctx context.Context, kubeSystemUUID string, entityID string) ([]metadataenrichment.Rule, error)
	// CreateOrUpdateKubernetesSetting returns the object ID of the created k8s settings.
	CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel, kubeSystemUUID, scope string) (string, error)
	// CreateOrUpdateKubernetesAppSetting returns the object ID of the created k8s app settings.
	CreateOrUpdateKubernetesAppSetting(ctx context.Context, scope string) (string, error)
	// CreateLogMonitoringSetting returns the object ID of the created logmonitoring settings.
	CreateLogMonitoringSetting(ctx context.Context, scope, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error)
	// GetKSPMSettings returns the settings response with the number of settings objects.
	GetKSPMSettings(ctx context.Context, monitoredEntity string) (GetSettingsResponse, error)
	// CreateKSPMSetting returns the object ID of the created kspm settings.
	CreateKSPMSetting(ctx context.Context, monitoredEntity string, datasetPipelineEnabled bool) (string, error)
	// DeleteSettings deletes the settings for a monitored entity.
	DeleteSettings(ctx context.Context, settingsID string) error
}

// K8sClusterME is representing the relevant info for a Kubernetes Cluster Monitored Entity
type K8sClusterME struct {
	ID   string
	Name string
}

type GetSettingsResponse struct {
	TotalCount int                 `json:"totalCount"`
	Items      []KSPMSettingObject `json:"items"`
}

type KSPMSettingObject struct {
	ObjectID string            `json:"objectId"`
	Value    KspmSettingsValue `json:"value"`
}

type KspmSettingsValue struct {
	DatasetPipelineEnabled bool `json:"configurationDatasetPipelineEnabled"`
}

type getKubernetesObjectsResponse struct {
	Items      []kubernetesObject `json:"items"`
	TotalCount int                `json:"totalCount"`
}

func (r getKubernetesObjectsResponse) MarshalLog() any {
	data, err := json.Marshal(r)
	if err != nil {
		// fallback to printing the struct with default formatting
		return r
	}

	return string(data)
}

var _ logr.Marshaler = getKubernetesObjectsResponse{}

type kubernetesObject struct {
	Scope string                `json:"scope"`
	Value kubernetesObjectValue `json:"value"`
}

type postObjectsResponse struct {
	ObjectID string `json:"objectId"`
}

type postObjectsBody[T any] struct {
	SchemaID      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion"`
	Scope         string `json:"scope,omitempty"`
	Value         T      `json:"value"`
}

// As of 1.26 type deduction is not good enough to omit the type from struct initialization.
func newPostObjectsBody[T any](schemaID, schemaVersion, scope string, value T) []postObjectsBody[T] {
	return []postObjectsBody[T]{
		{
			SchemaID:      schemaID,
			SchemaVersion: schemaVersion,
			Scope:         scope,
			Value:         value,
		},
	}
}

// getObjectID gives back the ID of the first element of the post response.
// If there are 0 or multiple entries, it will error.
// We only create (post) Settings if they do not exist yet, so receiving back not exactly one object is a cause for alarm.
func getObjectID(response []postObjectsResponse) (string, error) {
	if len(response) != 1 {
		return "", notSingleEntryError(len(response))
	}

	return response[0].ObjectID, nil
}

type notSingleEntryError int

func (num notSingleEntryError) Error() string {
	return fmt.Sprintf("response is not containing exactly one entry, got %d entries", int(num))
}

type Client struct {
	apiClient core.APIClient
}

func NewClient(apiClient core.APIClient) *Client {
	return &Client{
		apiClient: apiClient,
	}
}

// GetK8sClusterME returns the Kubernetes Cluster Monitored Entity for the give kubernetes cluster.
// Uses the `settings.read` scope to list the `builtin:cloud.kubernetes` settings.
//   - Only 1 such setting exists per tenant per kubernetes cluster
//   - The `scope` for the setting is the ID (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) of the Kubernetes Cluster Monitored Entity
//   - The `label` of the setting is the Name (example: my-dynakube) of the Kubernetes Cluster Monitored Entity
//
// In case 0 settings are found, so no Kubernetes Cluster Monitored Entity exists, we return an empty object, without an error.
func (c *Client) GetK8sClusterME(ctx context.Context, kubeSystemUUID string) (K8sClusterME, error) {
	if kubeSystemUUID == "" {
		return K8sClusterME{}, errMissingKubeSystemUUID
	}

	var response getKubernetesObjectsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "true",
			pageSizeQueryParam:     entitiesPageSize,
			schemaIDsQueryParam:    kubernetesSettingsSchemaID,
			fieldsQueryParam:       kubernetesSettingsNeededFields,
			filterQueryParam:       fmt.Sprintf("value.clusterId='%s'", kubeSystemUUID),
		}).
		Execute(&response)
	if err != nil {
		return K8sClusterME{}, fmt.Errorf("get k8s monitored entity: %w", err)
	}

	if len(response.Items) == 0 {
		log.Info("no kubernetes settings object according to API", "resp", response)

		return K8sClusterME{}, nil
	}

	return K8sClusterME{
		ID:   response.Items[0].Scope,
		Name: response.Items[0].Value.Label,
	}, nil
}

// GetSettingsForMonitoredEntity returns the settings response with the number of settings objects.
func (c *Client) GetSettingsForMonitoredEntity(ctx context.Context, monitoredEntity K8sClusterME, schemaID string) (GetSettingsResponse, error) {
	if monitoredEntity.ID == "" {
		return GetSettingsResponse{}, nil
	}

	var response GetSettingsResponse

	err := c.apiClient.GET(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			schemaIDsQueryParam: schemaID,
			scopesQueryParam:    monitoredEntity.ID,
		}).
		Execute(&response)
	if err != nil {
		return GetSettingsResponse{}, fmt.Errorf("get monitored entity settings: %w", err)
	}

	return response, nil
}

// DeleteSettings deletes the settings using the settings object ID.
func (c *Client) DeleteSettings(ctx context.Context, objectID string) error {
	if objectID == "" {
		return fmt.Errorf("cannot delete settings: no settings ID provided")
	}

	err := c.apiClient.DELETE(ctx, path.Join(ObjectsPath, objectID)).
		Execute(nil)
	if err != nil {
		return fmt.Errorf("delete monitored entity settings: %w", err)
	}

	return nil
}
