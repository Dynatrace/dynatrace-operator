// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// Package settings implements a client for the settings API, in either its gen2 (`/v2/settings/...`)
// or gen3 (`/platform/settings/v1/...`) form. See objectapi.go for how the two coexist.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/go-logr/logr"
)

// entitiesPageSize is only ever used by GetK8sClusterME's gen2 fallback scan.
const entitiesPageSize = "500"

var (
	errMissingKubeSystemUUID = errors.New("no kube-system namespace UUID given")
	errDeleteSettings        = errors.New("delete monitored entity settings failed")
	errNoSettingsIDProvided  = errors.New("no settings ID provided")
)

type Client interface {
	// GetK8sClusterME resolves the Kubernetes Cluster Monitored Entity for the cluster identified by
	// registration.EntityID, returning registration with EntityScope/EntityLabel
	// filled in once Dynatrace has resolved them.
	//   - Only 1 such setting exists per tenant per kubernetes cluster
	//   - EntityScope is the ID (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) of the Kubernetes Cluster Monitored Entity
	//   - EntityLabel is the display name (example: my-dynakube) of the Kubernetes Cluster Monitored Entity
	//
	// In case the setting does not exist (yet), so no Kubernetes Cluster Monitored Entity exists, registration is
	// returned unchanged, without an error.
	GetK8sClusterME(ctx context.Context, registration K8sClusterRegistration) (K8sClusterRegistration, error)
	// GetSettingsForMonitoredEntity returns the settings response with the number of settings objects and their values.
	GetSettingsForMonitoredEntity(ctx context.Context, registration K8sClusterRegistration, schemaID string) (TotalCountSettingsResponse, error)
	// GetSettingsForLogModule returns the settings response with the number of settings objects and their values.
	GetSettingsForLogModule(ctx context.Context, registration K8sClusterRegistration) (TotalCountSettingsResponse, error)
	// GetRules returns metadata enrichment rules.
	GetRules(ctx context.Context, registration K8sClusterRegistration) ([]metadataenrichment.Rule, error)
	// CreateOrUpdateKubernetesSetting returns the object ID of the created k8s settings.
	CreateOrUpdateKubernetesSetting(ctx context.Context, clusterLabel string, registration K8sClusterRegistration) (string, error)
	// CreateOrUpdateKubernetesAppSetting returns the object ID of the created k8s app settings.
	CreateOrUpdateKubernetesAppSetting(ctx context.Context, registration K8sClusterRegistration) (string, error)
	// CreateLogMonitoringSetting returns the object ID of the created logmonitoring settings.
	CreateLogMonitoringSetting(ctx context.Context, registration K8sClusterRegistration, clusterName string, matchers []logmonitoring.IngestRuleMatchers) (string, error)
	// GetKSPMSettings returns the settings response with the number of settings objects and their values.
	GetKSPMSettings(ctx context.Context, registration K8sClusterRegistration) (KSPMSettingsResponse, error)
	// CreateKSPMSetting returns the object ID of the created kspm settings.
	CreateKSPMSetting(ctx context.Context, registration K8sClusterRegistration, datasetPipelineEnabled bool) (string, error)
	// GetEnrichmentRuleObjects returns the list of enrichment rule settings objects (with objectIds) for the given cluster.
	// Only intended for e2e tests, where the number of rules is small. Does not handle pagination.
	GetEnrichmentRuleObjects(ctx context.Context, registration K8sClusterRegistration) ([]EnrichmentRuleObject, error)
	// GetLegacyEnrichmentRuleObjects returns enrichment rule settings objects for the legacy schema (builtin:kubernetes.generic.metadata.enrichment).
	// Only intended for e2e tests, where the number of rules is small. Does not handle pagination.
	GetLegacyEnrichmentRuleObjects(ctx context.Context, registration K8sClusterRegistration) ([]EnrichmentRuleObject, error)
	// CreateEnrichmentRuleObject creates a settings object for the builtin:ingest.enrichment.config schema.
	CreateEnrichmentRuleObject(ctx context.Context, registration K8sClusterRegistration, rules ...metadataenrichment.Rule) ([]string, error)
	// CreateLegacyEnrichmentRuleObject creates a settings object for the builtin:kubernetes.generic.metadata.enrichment schema.
	CreateLegacyEnrichmentRuleObject(ctx context.Context, registration K8sClusterRegistration, rules ...metadataenrichment.Rule) ([]string, error)
	// DeleteSettings deletes the settings for a monitored entity.
	DeleteSettings(ctx context.Context, settingsID string) error
}

// K8sClusterRegistration groups the identifiers a settings call needs to address a cluster.
// EntityID is known as soon as the operator can reach the cluster; EntityScope and EntityLabel are only
// known once Dynatrace has resolved the cluster's Monitored Entity.
//
// Once EntityScope is known, it is reused directly as the scope of settings calls (this is also how gen2
// always worked). Before that (i.e. before the Kubernetes Cluster Monitored Entity has been resolved at
// least once), gen3 has no unscoped writes, so it falls back to the Smartscape cluster-UID lookup scope
// derived from EntityID instead. See objectAPI.scope.
type K8sClusterRegistration struct {
	EntityID    string
	EntityScope string
	EntityLabel string
}

type TotalCountSettingsResponse struct {
	TotalCount int `json:"totalCount"`
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

// Generation selects which generation of the settings API a Client talks to.
type Generation int

const (
	Gen2 Generation = iota
	Gen3
)

type ClientImpl struct {
	api objectAPI
}

// NewClient creates a settings client for the given generation of the settings API. apiClient must
// already be scoped to that generation's base URL: gen2 is served under the classic "/api" host, gen3
// under the tenant's "*.apps.*" host.
func NewClient(apiClient core.Client, generation Generation) Client {
	var api objectAPI
	if generation == Gen3 {
		api = newGen3ObjectAPI(apiClient)
	} else {
		api = newGen2ObjectAPI(apiClient)
	}

	return &ClientImpl{api: api}
}

// GetK8sClusterME resolves the Kubernetes Cluster Monitored Entity for the cluster identified by
// registration.EntityID, returning registration with EntityScope/EntityLabel
// filled in once Dynatrace has resolved them.
//   - Only 1 such setting exists per tenant per kubernetes cluster
//   - EntityScope is the ID (example: KUBERNETES_CLUSTER-A1234567BCD8EFGH) of the Kubernetes Cluster Monitored Entity
//   - EntityLabel is the display name (example: my-dynakube) of the Kubernetes Cluster Monitored Entity
//
// In case the setting does not exist (yet), so no Kubernetes Cluster Monitored Entity exists, registration is
// returned unchanged, without an error.
func (c *ClientImpl) GetK8sClusterME(ctx context.Context, registration K8sClusterRegistration) (K8sClusterRegistration, error) {
	ctx, log := logd.NewFromContext(ctx, "dtclient-settings")

	if registration.EntityID == "" {
		return registration, errMissingKubeSystemUUID
	}

	var response getKubernetesObjectsResponse

	err := c.api.listObjects(ctx, objectsCollection, listParams{
		SchemaID:     KubernetesSettingsSchemaID,
		Scope:        c.api.scope(registration),
		Filter:       fmt.Sprintf("value.clusterId='%s'", registration.EntityID),
		AddFields:    kubernetesSettingsNeededFields,
		PageSize:     entitiesPageSize,
		ValidateOnly: true,
	}, &response)
	if err != nil {
		return registration, fmt.Errorf("get k8s monitored entity: %w", err)
	}

	if len(response.Items) == 0 {
		log.Info("no kubernetes settings object according to API", "resp", response)

		return registration, nil
	}

	registration.EntityScope = response.Items[0].Scope
	registration.EntityLabel = response.Items[0].Value.Label

	return registration, nil
}

// GetSettingsForMonitoredEntity returns the settings response with the number of settings objects.
func (c *ClientImpl) GetSettingsForMonitoredEntity(ctx context.Context, registration K8sClusterRegistration, schemaID string) (TotalCountSettingsResponse, error) {
	if registration.EntityScope == "" {
		return TotalCountSettingsResponse{}, nil
	}

	var response TotalCountSettingsResponse

	err := c.api.listObjects(ctx, objectsCollection, listParams{
		SchemaID:     schemaID,
		Scope:        c.api.scope(registration),
		ValidateOnly: true,
	}, &response)
	if err != nil {
		return TotalCountSettingsResponse{}, fmt.Errorf("get monitored entity settings: %w", err)
	}

	return response, nil
}

// DeleteSettings deletes the settings using the settings object ID.
func (c *ClientImpl) DeleteSettings(ctx context.Context, objectID string) error {
	if objectID == "" {
		return errNoSettingsIDProvided
	}

	if err := c.api.deleteObject(ctx, objectID); err != nil {
		return fmt.Errorf("%w: %w", errDeleteSettings, err)
	}

	return nil
}
