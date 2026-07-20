// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
)

// +kubebuilder:object:generate=true

type Status struct {
	status.VersionStatus `json:",inline"`

	// Information about KubernetesMonitoring's connections.
	ConnectionInfo communication.ConnectionInfo `json:"connectionInfo,omitempty,omitzero"`
}

func (s *Status) IsZero() bool {
	return s.VersionStatus.IsZero() && s.ConnectionInfo == communication.ConnectionInfo{}
}
