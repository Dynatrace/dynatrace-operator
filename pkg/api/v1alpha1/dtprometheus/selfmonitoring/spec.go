// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

// SelfMonitoring wraps the self-monitoring Spec together with the owning
// DtPrometheus name so derived state (such as Kubernetes resource names) can be
// computed.
type SelfMonitoring struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

// Spec toggles the optional self-monitoring collector, which ships the stack's
// own telemetry to Dynatrace. The collector's configuration (replicas, image,
// resources, scheduling, ...) lives on the referenced DynaKube.
type Spec struct {
	// Ships the stack's own telemetry to Dynatrace. Set to false to opt out.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`
}
