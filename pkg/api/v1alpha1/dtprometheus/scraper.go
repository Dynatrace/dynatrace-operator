// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScraperNameSuffix is appended to the owning DTPrometheus name to derive the base
// name of the scraper's Kubernetes resources.
const ScraperNameSuffix = "-scraper"

// +kubebuilder:object:generate=false

// Scraper wraps the scraper Spec together with the owning DTPrometheus name so
// derived state (such as Kubernetes resource names) can be computed.
type Scraper struct {
	*ScraperSpec

	name string
}

// ScraperSpec configures the scraper pool (tier 1): a Deployment of OTel Collectors
// that scrape their assigned targets and forward OTLP to the gateway pool.
type ScraperSpec struct {
	PodSpec `json:",inline"`

	// Default interval to poll target allocator for scrape targets.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="60s"
	// +kubebuilder:validation:Format=duration
	TargetsPollInterval metav1.Duration `json:"targetsPollInterval,omitempty"`

	// Deployment update strategy for the scraper pool.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DeploymentStrategy `json:"updateStrategy,omitzero"`
}

// NewScraper wraps the given Spec together with the owning DTPrometheus name.
func NewScraper(spec *ScraperSpec, name string) *Scraper {
	return &Scraper{
		ScraperSpec: spec,
		name:        name,
	}
}

// GetDeploymentName returns the base name for the scraper's Deployment.
func (s *Scraper) GetDeploymentName() string {
	return s.name + ScraperNameSuffix
}
