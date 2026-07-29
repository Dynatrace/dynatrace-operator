// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dtprometheus/common"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetAllocator wraps the Target Allocator Spec together with the owning
// DtPrometheus name so derived state (such as Kubernetes resource names) can be
// computed.
type TargetAllocator struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

// Spec configures the Target Allocator Deployment, which holds all Prometheus
// service discovery metadata and distributes scrape targets across the scraper pool.
type Spec struct {
	common.Spec `json:",inline"`

	// Default scrape interval applied to all CRD-based targets when not
	// overridden in the individual ServiceMonitor / PodMonitor / ScrapeConfig.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="60s"
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
