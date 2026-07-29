// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dtprometheus/common"
	appsv1 "k8s.io/api/apps/v1"
)

// SelfMonitoring wraps the self-monitoring Spec together with the owning
// DtPrometheus name so derived state (such as Kubernetes resource names) can be
// computed.
type SelfMonitoring struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

// Spec configures the optional self-monitoring collector, which ships the stack's
// own telemetry to Dynatrace. It carries the same configuration as the scraper.
type Spec struct {
	// Set to true to opt in to shipping the stack's own telemetry to Dynatrace.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	common.Spec `json:",inline"`

	// Deployment update strategy for the self-monitoring collector.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DeploymentStrategy `json:"updateStrategy,omitzero"`
}
