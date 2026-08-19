// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1

package dtprometheus

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DTPrometheusSpec defines the desired state of DTPrometheus.
type DTPrometheusSpec struct { //nolint:revive
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
	DynatracePreset *DynatracePresetSpec `json:"dynatracePreset,omitzero"`

	// Configures the Target Allocator, which holds all Prometheus service
	// discovery metadata and distributes scrape targets across the scraper pool.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={}
	TargetAllocator TargetAllocatorSpec `json:"targetAllocator"`

	// Configures the scraper pool (tier 1): a Deployment of OTel Collectors that
	// scrape their assigned targets and forward OTLP to the gateway pool.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={}
	Scraper ScraperSpec `json:"scraper"`

	// Configures the gateway pool (tier 2): a StatefulSet of OTel Collectors that
	// run stateful processors and export to Dynatrace via OTLP/HTTP.
	// +kubebuilder:validation:Optional
	Gateway GatewaySpec `json:"gateway,omitzero"`

	// Overrides the default registry from which Dynatrace images are pulled.
	// +kubebuilder:validation:Optional
	PublicRegistryOverride string `json:"publicRegistryOverride,omitempty"`
}

// DynatracePresetSpec toggles the operator-managed annotation-based ScrapeConfig.
type DynatracePresetSpec struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dtprometheuses,scope=Namespaced,categories=dynatrace,shortName={dtp,dtps}
// +kubebuilder:printcolumn:name="DynaKube",type=string,JSONPath=`.spec.dynaKubeName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DTPrometheus is the Schema for the DTPrometheus API.
type DTPrometheus struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +kubebuilder:validation:Optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of DTPrometheus
	// +kubebuilder:validation:Required
	Spec DTPrometheusSpec `json:"spec"`

	// status defines the observed state of DTPrometheus
	// +kubebuilder:validation:Optional
	Status DTPrometheusStatus `json:"status,omitzero"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true

// DTPrometheusList contains a list of DTPrometheus.
type DTPrometheusList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DTPrometheus `json:"items"`
}
