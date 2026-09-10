// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1

package dtprometheus

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusMonitoringSpec defines the desired state of PrometheusMonitoring.
type PrometheusMonitoringSpec struct { //nolint:revive
	// Name of the DynaKube in the same namespace that provides all connection
	// settings (apiUrl, tokens, proxy, networkZone, trustedCAs, ActiveGate, etc.).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DynaKubeRef string `json:"dynaKubeRef"`

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
	// +kubebuilder:default={}
	Gateway GatewaySpec `json:"gateway"`

	// Overrides the default registry from which Dynatrace images are pulled.
	// +kubebuilder:validation:Optional
	PublicRegistryOverride string `json:"publicRegistryOverride,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=prometheusmonitorings,scope=Namespaced,categories=dynatrace,shortName={pm,pms}
// +kubebuilder:printcolumn:name="DynaKube",type=string,JSONPath=`.spec.dynaKubeRef`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PrometheusMonitoring is the Schema for the PrometheusMonitoring API.
type PrometheusMonitoring struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +kubebuilder:validation:Optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PrometheusMonitoring
	// +kubebuilder:validation:Required
	Spec PrometheusMonitoringSpec `json:"spec"`

	// status defines the observed state of PrometheusMonitoring
	// +kubebuilder:validation:Optional
	Status PrometheusMonitoringStatus `json:"status,omitzero"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true

// PrometheusMonitoringList contains a list of PrometheusMonitoring.
type PrometheusMonitoringList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PrometheusMonitoring `json:"items"`
}
