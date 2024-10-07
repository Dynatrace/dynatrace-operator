package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
)

// +kubebuilder:object:generate=true
type Status struct {
	status.VersionStatus `json:",inline"`

	// Information about Active Gate's connections
	ConnectionInfo common.ConnectionInfo `json:"connectionInfoStatus,omitempty"`

	// The ClusterIPs set by Kubernetes on the ActiveGate Service created by the Operator
	ServiceIPs []string `json:"serviceIPs,omitempty"`
}
