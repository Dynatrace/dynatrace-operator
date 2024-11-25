package dynatrace

import (
	"bytes"
	"context"
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
	*MonitoringSettings
	Label            string `json:"label"`
	ClusterId        string `json:"clusterId"`
	ClusterIdEnabled bool   `json:"clusterIdEnabled"`
	Enabled          bool   `json:"enabled"`
}

type MonitoringSettings struct {
	CloudApplicationPipelineEnabled bool `json:"cloudApplicationPipelineEnabled"`
	OpenMetricsPipelineEnabled      bool `json:"openMetricsPipelineEnabled"`
	EventProcessingActive           bool `json:"eventProcessingActive"`
	EventProcessingV2Active         bool `json:"eventProcessingV2Active"`
	FilterEvents                    bool `json:"filterEvents"`
}

type postKubernetesSettingsBody struct {
	Value         any    `json:"value"`
	SchemaId      string `json:"schemaId"`
	SchemaVersion string `json:"schemaVersion"`
	Scope         string `json:"scope,omitempty"`
}

type postKubernetesAppSettings struct {
	KubernetesAppOptions kubernetesAppOptionsSettings `json:"kubernetesAppOptions"`
}
type kubernetesAppOptionsSettings struct {
	EnableKubernetesApp bool `json:"enableKubernetesApp"`
}

const (
	KubernetesSettingsSchemaId                  = "builtin:cloud.kubernetes"
	AppTransitionSchemaId                       = "builtin:app-transition.kubernetes"
	schemaVersionV1                             = "1.0.27"
	hierarchicalMonitoringSettingsSchemaVersion = "3.0.0"
	appTransitionSchemaVersion                  = "1.0.1"
)

func (dtc *dynatraceClient) performCreateOrUpdateKubernetesSetting(ctx context.Context, body []postKubernetesSettingsBody) (string, error) { //nolint:dupl
	bodyData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := createBaseRequest(ctx, dtc.getSettingsUrl(false), http.MethodPost, dtc.apiToken, bytes.NewReader(bodyData))
	if err != nil {
		return "", err
	}

	res, err := dtc.httpClient.Do(req)
	defer utils.CloseBodyAfterRequest(res)

	if err != nil {
		return "", fmt.Errorf("error making post request to dynatrace api: %w", err)
	}

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

	settings := createBaseKubernetesSettings(postK8sSettings, KubernetesSettingsSchemaId, schemaVersionV1, scope)

	return []postKubernetesSettingsBody{settings}
}

func createV3KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope string) []postKubernetesSettingsBody {
	settings := createBaseKubernetesSettings(
		createPostKubernetesSettings(clusterLabel, kubeSystemUUID),
		KubernetesSettingsSchemaId,
		hierarchicalMonitoringSettingsSchemaVersion,
		scope)
	settings.SchemaVersion = hierarchicalMonitoringSettingsSchemaVersion

	return []postKubernetesSettingsBody{settings}
}

func (dtc *dynatraceClient) CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel, kubeSystemUUID, scope string) (string, error) {
	if kubeSystemUUID == "" {
		return "", errors.New("no kube-system namespace UUID given")
	}

	body := createV3KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope)

	objectId, err := dtc.performCreateOrUpdateKubernetesSetting(ctx, body)
	if err != nil {
		if strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
			body = createV1KubernetesSettingsBody(clusterLabel, kubeSystemUUID, scope)

			return dtc.performCreateOrUpdateKubernetesSetting(ctx, body)
		} else {
			return "", err
		}
	}

	return objectId, nil
}

func (dtc *dynatraceClient) CreateOrUpdateKubernetesAppSetting(ctx context.Context, scope string) (string, error) {
	settings := createBaseKubernetesSettings(postKubernetesAppSettings{
		kubernetesAppOptionsSettings{
			EnableKubernetesApp: true,
		},
	}, AppTransitionSchemaId, appTransitionSchemaVersion, scope)

	objectId, err := dtc.performCreateOrUpdateKubernetesSetting(ctx, []postKubernetesSettingsBody{settings})
	if err != nil {
		return "", err
	}

	return objectId, nil
}
