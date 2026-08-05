// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TargetAllocatorNameSuffix is appended to the owning DTPrometheus name to derive
	// the base name of the Target Allocator's Kubernetes resources.
	TargetAllocatorNameSuffix = "-target-allocator"

	// DefaultScrapeCRSelectorLabel is the label key the Target Allocator matches
	// on Prometheus CRDs when no explicit scrapeCRSelector is configured.
	DefaultScrapeCRSelectorLabel = "prometheus.dynatrace.com"
)

// +kubebuilder:object:generate=false

// TargetAllocator wraps the Target Allocator Spec together with the owning
// DTPrometheus name so derived state (such as Kubernetes resource names) can be
// computed.
type TargetAllocator struct {
	*TargetAllocatorSpec

	name string
}

// TargetAllocatorSpec configures the Target Allocator Deployment, which holds all
// Prometheus service discovery metadata and distributes scrape targets across the
// scraper pool.
type TargetAllocatorSpec struct {
	PodSpec `json:",inline"`

	// Default interval used by the target allocator to look for Prometheus CRs
	// (ServiceMonitor, PodMonitor, ScrapeConfig, Probe).
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="60s"
	// +kubebuilder:validation:Format=duration
	ScrapeInterval string `json:"scrapeInterval,omitempty"`

	// Label selector applied to all Prometheus Operator CRDs (ServiceMonitor,
	// PodMonitor, ScrapeConfig, Probe). The TA only picks up CRDs whose labels
	// match. When omitted, the operator defaults to matching
	// prometheus.dynatrace.com: "true". An empty selector {} matches all CRDs.
	// +kubebuilder:validation:Optional
	ScrapeCRSelector *metav1.LabelSelector `json:"scrapeCRSelector,omitempty"`

	// Restricts which namespaces the TA watches for CRDs. An empty selector {}
	// matches all namespaces.
	// +kubebuilder:validation:Optional
	ScrapeCRNamespaceSelector *metav1.LabelSelector `json:"scrapeCRNamespaceSelector,omitempty"`

	// Deployment update strategy for the Target Allocator.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DeploymentStrategy `json:"updateStrategy,omitzero"`
}

// NewTargetAllocator wraps the given Spec together with the owning DTPrometheus name.
func NewTargetAllocator(spec *TargetAllocatorSpec, name string) *TargetAllocator {
	return &TargetAllocator{
		TargetAllocatorSpec: spec,
		name:                name,
	}
}

// GetDeploymentName returns the base name for the Target Allocator's Deployment.
func (ta *TargetAllocator) GetDeploymentName() string {
	return ta.name + TargetAllocatorNameSuffix
}

// GetScrapeCRSelector returns the configured CRD label selector. When none is set
// it defaults to matching prometheus.dynatrace.com: "true", as documented on the field.
func (ta *TargetAllocator) GetScrapeCRSelector() *metav1.LabelSelector {
	if ta.ScrapeCRSelector != nil {
		return ta.ScrapeCRSelector
	}

	return &metav1.LabelSelector{
		MatchLabels: map[string]string{DefaultScrapeCRSelectorLabel: "true"},
	}
}
