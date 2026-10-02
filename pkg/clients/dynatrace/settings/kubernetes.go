// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

	kubernetesSettingsNeededFields = "value,scope"

	k8sClusterUIDScopePrefix = "k8s.cluster.uid:"
)

// k8sClusterUIDScope returns the virtual scope that the builtin:cloud.kubernetes setting is created
// and queried under on gen3. Once persisted, Dynatrace resolves this virtual scope to the actual
// Kubernetes Cluster Monitored Entity ID.
func k8sClusterUIDScope(kubeSystemUUID string) string {
	return k8sClusterUIDScopePrefix + kubeSystemUUID
}

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
// The scope used depends on the API generation: gen2 uses an empty scope and lets the backend create
// the Monitored Entity asynchronously; gen3 has no unscoped writes, so it uses the Smartscape
// cluster-UID lookup scope, which Dynatrace resolves to the Monitored Entity ID once persisted.
func (c *ClientImpl) CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel string, registration K8sClusterRegistration) (string, error) {
	if registration.EntityID == "" {
		return "", errMissingKubeSystemUUID
	}

	scope := c.api.scope(registration)

	objectID, err := c.performCreateOrUpdateKubernetesSetting(ctx, KubernetesSettingsSchemaID, hierarchicalMonitoringSettingsSchemaVersion, scope,
		newKubernetesObjectValue(clusterLabel, registration.EntityID))
	if err != nil {
		if !core.IsNotFound(err) {
			return "", err
		}

		v1Value := newKubernetesObjectValue(clusterLabel, registration.EntityID)
		v1Value.monitoringSettings = &monitoringSettings{CloudApplicationPipelineEnabled: true}

		return c.performCreateOrUpdateKubernetesSetting(ctx, KubernetesSettingsSchemaID, schemaVersionV1, scope, v1Value)
	}

	return objectID, nil
}

// CreateOrUpdateKubernetesAppSetting returns the object ID of the created k8s app settings.
func (c *ClientImpl) CreateOrUpdateKubernetesAppSetting(ctx context.Context, registration K8sClusterRegistration) (string, error) {
	return c.performCreateOrUpdateKubernetesSetting(ctx, AppTransitionSchemaID, appTransitionSchemaVersion, c.api.scope(registration),
		kubernetesAppObjectValue{
			KubernetesAppOptions: kubernetesAppOptions{
				EnableKubernetesApp: true,
			},
		})
}

func (c *ClientImpl) performCreateOrUpdateKubernetesSetting(ctx context.Context, schemaID, schemaVersion, scope string, value any) (string, error) {
	ids, err := c.api.createObject(ctx, schemaID, schemaVersion, scope, value)
	if err != nil {
		return "", fmt.Errorf("create kubernetes setting: %w", err)
	}

	return singleID(ids)
}

func newKubernetesObjectValue(clusterLabel, kubeSystemUUID string) kubernetesObjectValue {
	return kubernetesObjectValue{
		Label:            clusterLabel,
		ClusterID:        kubeSystemUUID,
		ClusterIDEnabled: true,
		Enabled:          true,
	}
}
