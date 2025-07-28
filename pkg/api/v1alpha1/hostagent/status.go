package hostagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentPhase string

const (
	Running   DeploymentPhase = "Running"
	Deploying DeploymentPhase = "Deploying"
	Error     DeploymentPhase = "Error"
)

// Status defines the observed state of ActiveGate.
type Status struct {
	// Defines the current state (Running, Updating, Error, ...)
	DeploymentPhase DeploymentPhase `json:"phase,omitempty"`

	ConnectionInfo communication.ConnectionInfo `json:"connectionInfo,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SetPhase sets the status phase on the ActiveGate object.
func (s *Status) SetPhase(phase DeploymentPhase) bool {
	upd := phase != s.DeploymentPhase
	s.DeploymentPhase = phase

	return upd
}
