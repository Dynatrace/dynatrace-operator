package edgeconnect

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EdgeConnectStatus defines the observed state of EdgeConnect.
type EdgeConnectStatus struct { //nolint:revive
	// Defines the current state (Running, Updating, Error, ...)
	DeploymentPhase status.DeploymentPhase `json:"phase,omitempty"`

	// Version used for the Edgeconnect image
	Version status.VersionStatus `json:"version,omitempty"`

	// Indicates when the resource was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// kube-system namespace uid
	KubeSystemUID string `json:"kubeSystemUID,omitempty"`

	// Conditions includes status about the current state of the instance
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the EdgeConnect object.
func (dk *EdgeConnectStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.DeploymentPhase
	dk.DeploymentPhase = phase

	return upd
}

func (ec *EdgeConnect) UpdateStatus(ctx context.Context, client client.Client) error {
	ec.Status.UpdatedTimestamp = metav1.Now()
	err := client.Status().Update(ctx, ec)

	return errors.WithStack(err)
}
