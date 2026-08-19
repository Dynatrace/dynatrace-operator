// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DTPrometheusStatus defines the observed state of DTPrometheus.
type DTPrometheusStatus struct { //nolint:revive
	// Defines the current state (Running, Deploying, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	Gateway GatewayStatus `json:"gateway,omitempty"`

	// Conditions includes status about the current state of the instance
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type GatewayStatus struct {
	// Image URI of the gateway currently deployed.
	ResolvedImage string `json:"image,omitempty"`
}

// SetPhase sets the status phase on the DTPrometheus object.
func (dtps *DTPrometheusStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dtps.Phase
	dtps.Phase = phase

	return upd
}
