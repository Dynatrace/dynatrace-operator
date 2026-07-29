// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
)

// +kubebuilder:object:generate=true

type Status struct {
	// Information about the version of KubernetesMonitoring that is meant to be deployed.
	Version string `json:"version,omitempty"`

	// Information about KubernetesMonitoring's connections.
	// +kubebuilder:validation:Optional
	ConnectionInfo communication.ConnectionInfo `json:"connectionInfo,omitzero"`
}

func (s *Status) IsZero() bool {
	return s.Version == "" && s.ConnectionInfo == communication.ConnectionInfo{}
}
