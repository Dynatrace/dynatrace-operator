// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DtPrometheusStatus defines the observed state of DtPrometheus.
type DtPrometheusStatus struct { //nolint:revive
	// Defines the current state (Running, Deploying, Error, ...)
	// +kubebuilder:validation:Optional
	DeploymentPhase status.DeploymentPhase `json:"phase,omitempty"`

	// Indicates when the resource was last updated
	// +kubebuilder:validation:Optional
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	// +kubebuilder:validation:Optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the DtPrometheus object.
func (dtps *DtPrometheusStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dtps.DeploymentPhase
	dtps.DeploymentPhase = phase

	return upd
}
