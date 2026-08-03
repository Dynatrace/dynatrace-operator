// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1

package dtprometheus

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DtPrometheusSpec defines the desired state of DtPrometheus.
type DtPrometheusSpec struct { //nolint:revive
	// Name of the DynaKube in the same namespace that provides all connection
	// settings (apiUrl, tokens, proxy, networkZone, trustedCAs, ActiveGate, etc.).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DynaKubeName string `json:"dynaKubeName"`

	// When enabled, the operator creates a built-in ScrapeConfig that discovers
	// pods annotated with metrics.dynatrace.com/scrape: "true" and reads the
	// port, path, and secure annotations to construct scrape targets.
	// This provides backwards compatibility with the ActiveGate Kubernetes
	// module annotation-based discovery workflow.
	// +kubebuilder:validation:Optional
	DynatracePreset DynatracePresetSpec `json:"dynatracePreset,omitzero"`

	// Controls TLS for internal component communication
	// (TA -> scraper HTTPS, scraper -> gateway OTLP/TLS).
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={}
	// +nullable
	TLS TLSSpec `json:"tls"`

	// Deploys a dedicated self-monitoring collector that ships the stack's own
	// telemetry to Dynatrace. Enabled by default; set to null to opt out.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={}
	// +nullable
	SelfMonitoring *SelfMonitoringSpec `json:"selfMonitoring"`

	// Configures the Target Allocator, which holds all Prometheus service
	// discovery metadata and distributes scrape targets across the scraper pool.
	// +kubebuilder:validation:Optional
	TargetAllocator TargetAllocatorSpec `json:"targetAllocator,omitzero"`

	// Configures the scraper pool (tier 1): a Deployment of OTel Collectors that
	// scrape their assigned targets and forward OTLP to the gateway pool.
	// +kubebuilder:validation:Optional
	Scraper ScraperSpec `json:"scraper,omitzero"`

	// Configures the gateway pool (tier 2): a StatefulSet of OTel Collectors that
	// run stateful processors and export to Dynatrace via OTLP/HTTP.
	// +kubebuilder:validation:Optional
	Gateway GatewaySpec `json:"gateway,omitzero"`
}

// DynatracePresetSpec toggles the operator-managed annotation-based ScrapeConfig.
type DynatracePresetSpec struct {
	// Enable the built-in annotation-based ScrapeConfig.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`
}

// TLSSpec configures operator-managed TLS between components.
type TLSSpec struct {
	// Set to false to disable operator-managed TLS. Use when a service mesh
	// (Istio, Linkerd) handles mTLS between components.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
}

// SelfMonitoringSpec toggles the optional self-monitoring collector, which ships
// the stack's own telemetry to Dynatrace. Its presence enables the collector; the
// collector's configuration (replicas, image, resources, scheduling, ...) lives on
// the referenced DynaKube.
type SelfMonitoringSpec struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dtprometheuses,scope=Namespaced,categories=dynatrace,shortName={dtp,dtps}
// +kubebuilder:printcolumn:name="DynaKube",type=string,JSONPath=`.spec.dynaKubeName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DtPrometheus is the Schema for the DtPrometheus API.
type DtPrometheus struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +kubebuilder:validation:Optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of DtPrometheus
	// +kubebuilder:validation:Required
	Spec DtPrometheusSpec `json:"spec"`

	// status defines the observed state of DtPrometheus
	// +kubebuilder:validation:Optional
	Status DtPrometheusStatus `json:"status,omitzero"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true

// DtPrometheusList contains a list of DtPrometheus.
type DtPrometheusList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DtPrometheus `json:"items"`
}
