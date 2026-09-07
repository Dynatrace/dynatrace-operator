// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
)

// +kubebuilder:object:generate=true

type Status struct {
	status.VersionStatus `json:",inline"`

	// Information about Active Gate's connections
	ConnectionInfo communication.ConnectionInfo `json:"connectionInfoStatus,omitempty"`

	// The ClusterIPs set by Kubernetes on the ActiveGate Service created by the Operator
	ServiceIPs []string `json:"serviceIPs,omitempty"`
}
