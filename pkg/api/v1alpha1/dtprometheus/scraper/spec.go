// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus/common"
	appsv1 "k8s.io/api/apps/v1"
)

// Scraper wraps the scraper Spec together with the owning DtPrometheus name so
// derived state (such as Kubernetes resource names) can be computed.
type Scraper struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

// Spec configures the scraper pool (tier 1): a Deployment of OTel Collectors that
// scrape their assigned targets and forward OTLP to the gateway pool.
type Spec struct {
	common.Spec `json:",inline"`

	// Deployment update strategy for the scraper pool.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DeploymentStrategy `json:"updateStrategy,omitzero"`
}
