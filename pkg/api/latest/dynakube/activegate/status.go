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

// IsZero reports whether every field is zero. It is required for the `omitzero`
// JSON tag: the promoted status.VersionStatus.IsZero() only inspects the version
// fields, so without this method omitzero would drop the whole status (including
// connection info) whenever the version happens to be unset.
func (ag *Status) IsZero() bool {
	return ag.VersionStatus.IsZero() &&
		ag.ConnectionInfo == communication.ConnectionInfo{} &&
		len(ag.ServiceIPs) == 0
}

// GetImage provides the image reference set in Status for the ActiveGate.
// Format: repo@sha256:digest.
func (ag *Status) GetImage() string {
	return ag.ImageID
}

// GetVersion provides version set in Status for the ActiveGate.
func (ag *Status) GetVersion() string {
	return ag.Version
}
