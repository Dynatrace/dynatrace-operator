// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NameSuffix is appended to the owning DtPrometheus name to derive the base
	// name of the Target Allocator's Kubernetes resources.
	NameSuffix = "-target-allocator"

	// DefaultScrapeInterval is applied when no scrape interval is configured.
	DefaultScrapeInterval = "60s"

	// DefaultScrapeCRSelectorLabel is the label key the Target Allocator matches
	// on Prometheus CRDs when no explicit scrapeCRSelector is configured.
	DefaultScrapeCRSelectorLabel = "prometheus.dynatrace.com"
)

// New wraps the given Spec together with the owning DtPrometheus name.
func New(spec *Spec, name string) *TargetAllocator {
	return &TargetAllocator{
		Spec: spec,
		name: name,
	}
}

// SetName sets the owning DtPrometheus name.
func (ta *TargetAllocator) SetName(name string) {
	ta.name = name
}

// GetName returns the base name for the Target Allocator's Kubernetes resources.
func (ta *TargetAllocator) GetName() string {
	return ta.name + NameSuffix
}

// GetScrapeInterval returns the configured scrape interval, or the default when unset.
func (ta *TargetAllocator) GetScrapeInterval() string {
	if ta.ScrapeInterval == "" {
		return DefaultScrapeInterval
	}

	return ta.ScrapeInterval
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

// GetScrapeCRNamespaceSelector returns the configured namespace selector. A nil or
// empty selector matches all namespaces.
func (ta *TargetAllocator) GetScrapeCRNamespaceSelector() *metav1.LabelSelector {
	return ta.ScrapeCRNamespaceSelector
}

// GetUpdateStrategy returns the Deployment update strategy for the Target Allocator.
func (ta *TargetAllocator) GetUpdateStrategy() appsv1.DeploymentStrategy {
	return ta.UpdateStrategy
}
