package settings

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
)

const (
	KubernetesSettingsSchemaID = "builtin:cloud.kubernetes"
	AppTransitionSchemaID      = "builtin:app-transition.kubernetes"

	schemaVersionV1                             = "1.0.27"
	hierarchicalMonitoringSettingsSchemaVersion = "3.0.0"
	appTransitionSchemaVersion                  = "1.0.1"
)

type kubernetesObjectValue struct {
	*monitoringSettings
	Label            string `json:"label"`
	ClusterID        string `json:"clusterId"`
	ClusterIDEnabled bool   `json:"clusterIdEnabled"`
	Enabled          bool   `json:"enabled"`
}

type monitoringSettings struct {
	CloudApplicationPipelineEnabled bool `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool `json:"openMetricsPipelineEnabled"`
	EventProcessingActive           bool `json:"eventProcessingActive"`
	EventProcessingV2Active         bool `json:"eventProcessingV2Active"`
	FilterEvents                    bool `json:"filterEvents"`
}

type kubernetesAppObjectValue struct {
	KubernetesAppOptions kubernetesAppOptions `json:"kubernetesAppOptions"`
}

type kubernetesAppOptions struct {
	EnableKubernetesApp bool `json:"enableKubernetesApp"`
}

// CreateOrUpdateKubernetesSetting returns the object ID of the created k8s settings.
func (c *Client) CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel, kubeSystemUUID, scope string) (string, error) {
	if kubeSystemUUID == "" {
		return "", errMissingKubeSystemUUID
	}

	body := v3KubernetesObjectBody(clusterLabel, kubeSystemUUID, scope)

	objectID, err := c.performCreateOrUpdateKubernetesSetting(ctx, body)
	if err != nil {
		if !core.IsNotFound(err) {
			return "", err
		}

		body = v1KubernetesObjectBody(clusterLabel, kubeSystemUUID, scope)

		return c.performCreateOrUpdateKubernetesSetting(ctx, body)
	}

	return objectID, nil
}

// CreateOrUpdateKubernetesAppSetting returns the object ID of the created k8s app settings.
func (c *Client) CreateOrUpdateKubernetesAppSetting(ctx context.Context, scope string) (string, error) {
	settings := newPostObjectsBody(
		AppTransitionSchemaID, appTransitionSchemaVersion, scope,
		kubernetesAppObjectValue{
			kubernetesAppOptions{
				EnableKubernetesApp: true,
			},
		},
	)

	objectID, err := c.performCreateOrUpdateKubernetesSetting(ctx, settings)
	if err != nil {
		return "", err
	}

	return objectID, nil
}

func (c *Client) performCreateOrUpdateKubernetesSetting(ctx context.Context, body any) (string, error) {
	var response []postObjectsResponse

	err := c.apiClient.POST(ctx, ObjectsPath).
		WithQueryParams(map[string]string{
			validateOnlyQueryParam: "false",
		}).
		WithJSONBody(body).
		Execute(&response)
	if err != nil {
		return "", fmt.Errorf("create kubernetes setting: %w", err)
	}

	if len(response) != 1 {
		return "", tooManyEntriesError(len(response))
	}

	return response[0].ObjectID, nil
}

func v1KubernetesObjectBody(clusterLabel, kubeSystemUUID, scope string) []postObjectsBody[kubernetesObjectValue] {
	settings := newKubernetesObjectValue(clusterLabel, kubeSystemUUID)
	settings.monitoringSettings = &monitoringSettings{
		CloudApplicationPipelineEnabled: true,
	}

	return newPostObjectsBody(KubernetesSettingsSchemaID, schemaVersionV1, scope, settings)
}

func v3KubernetesObjectBody(clusterLabel, kubeSystemUUID, scope string) []postObjectsBody[kubernetesObjectValue] {
	return newPostObjectsBody(
		KubernetesSettingsSchemaID,
		hierarchicalMonitoringSettingsSchemaVersion,
		scope,
		newKubernetesObjectValue(clusterLabel, kubeSystemUUID),
	)
}

func newKubernetesObjectValue(clusterLabel, kubeSystemUUID string) kubernetesObjectValue {
	return kubernetesObjectValue{
		Label:            clusterLabel,
		ClusterID:        kubeSystemUUID,
		ClusterIDEnabled: true,
		Enabled:          true,
	}
}
