// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DtPrometheusStatus defines the observed state of DtPrometheus.
type DtPrometheusStatus struct { //nolint:revive
	// Defines the current state (Running, Deploying, Error, ...)
	DeploymentPhase status.DeploymentPhase `json:"phase,omitempty"`

	// Indicates when the resource was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
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

// UpdateStatus stamps UpdatedTimestamp and persists the status subresource.
func (dtp *DtPrometheus) UpdateStatus(ctx context.Context, apiClient client.Client) error {
	dtp.Status.UpdatedTimestamp = metav1.Now()
	err := apiClient.Status().Update(ctx, dtp)

	return errors.WithStack(err)
}
