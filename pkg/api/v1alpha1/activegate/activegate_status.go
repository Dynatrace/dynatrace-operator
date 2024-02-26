package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ActiveGateStatus defines the observed state of ActiveGate.
type ActiveGateStatus struct { //nolint:revive
	// Defines the current state (Running, Updating, Error, ...)
	DeploymentPhase status.DeploymentPhase `json:"phase,omitempty"`

	// Indicates when the resource was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the ActiveGate object.
func (dk *ActiveGateStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.DeploymentPhase
	dk.DeploymentPhase = phase

	return upd
}

func (dk *ActiveGate) UpdateStatus(ctx context.Context, client client.Client) error {
	dk.Status.UpdatedTimestamp = metav1.Now()
	err := client.Status().Update(ctx, dk)

	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update activegate due to conflict", "name", dk.Name)

		return nil
	}

	return errors.WithStack(err)
}
