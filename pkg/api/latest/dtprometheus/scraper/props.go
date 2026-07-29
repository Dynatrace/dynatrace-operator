// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import appsv1 "k8s.io/api/apps/v1"

// NameSuffix is appended to the owning DtPrometheus name to derive the base name
// of the scraper's Kubernetes resources.
const NameSuffix = "-scraper"

// New wraps the given Spec together with the owning DtPrometheus name.
func New(spec *Spec, name string) *Scraper {
	return &Scraper{
		Spec: spec,
		name: name,
	}
}

// SetName sets the owning DtPrometheus name.
func (s *Scraper) SetName(name string) {
	s.name = name
}

// GetName returns the base name for the scraper's Kubernetes resources.
func (s *Scraper) GetName() string {
	return s.name + NameSuffix
}

// GetUpdateStrategy returns the Deployment update strategy for the scraper pool.
func (s *Scraper) GetUpdateStrategy() appsv1.DeploymentStrategy {
	return s.UpdateStrategy
}
