// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
)

// +kubebuilder:object:generate=true

type Status struct {
	// The resolved KubernetesMonitoring image that is currently deployed.
	// The JSON tag uses "image" (matching the Spec field name) while the Go field name is ResolvedImage to distinguish it from Spec.Image.
	ResolvedImage string `json:"image,omitempty"`

	// Information about KubernetesMonitoring's connections.
	// +kubebuilder:validation:Optional
	ConnectionInfo communication.ConnectionInfo `json:"connectionInfo,omitzero"`

	// TLSSecretHash contains the hash of the TLS certificate and key that is passed to both the Kubernetes Monitoring ActiveGate and Node-Configuration-Collector.
	// Meant to keep the two in sync.
	TLSSecretHash string `json:"tlsSecretHash,omitempty"`
}
