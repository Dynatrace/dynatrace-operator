// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube/oneagent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct { //nolint:revive

	// Observed state of OneAgent
	OneAgent oneagent.Status `json:"oneAgent,omitempty"`

	// Observed state of ActiveGate
	ActiveGate activegate.Status `json:"activeGate,omitempty"`

	// Observed state of Code Modules
	CodeModules oneagent.CodeModulesStatus `json:"codeModules,omitempty"`

	// Observed state of Metadata-Enrichment
	MetadataEnrichment MetadataEnrichmentStatus `json:"metadataEnrichment,omitempty"`

	// Observed state of KSPM
	KSPM kspm.Status `json:"kspm,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// ProxyURLHash is the hashed value of what is in spec.proxy.
	// Used for setting it as an annotation value for components that use the proxy.
	// This annotation will cause the component to be restarted if the proxy changes.
	ProxyURLHash string `json:"proxyURLHash,omitempty"`

	// Observed state of Dynatrace API
	DynatraceAPI DynatraceAPIStatus `json:"dynatraceApi,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// KubernetesClusterMEID contains the ID of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterMEID string `json:"kubernetesClusterMEID,omitempty"`

	// KubernetesClusterName contains the display name (also know as label) of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterName string `json:"kubernetesClusterName,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type DynatraceAPIStatus struct {
	// Time of the last token request
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`
}

type EnrichmentRuleType string

type MetadataEnrichmentStatus struct {
	Rules []EnrichmentRule `json:"rules,omitempty"`
}

type EnrichmentRule struct {
	Type   EnrichmentRuleType `json:"type,omitempty"`
	Source string             `json:"source,omitempty"`
	Target string             `json:"target,omitempty"`
}
