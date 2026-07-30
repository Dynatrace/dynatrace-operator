// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

// NameSuffix is appended to the owning DtPrometheus name to derive the base name
// of the self-monitoring collector's Kubernetes resources.
const NameSuffix = "-self-monitoring"

// New wraps the given Spec together with the owning DtPrometheus name.
func New(spec *Spec, name string) *SelfMonitoring {
	return &SelfMonitoring{
		Spec: spec,
		name: name,
	}
}

// SetName sets the owning DtPrometheus name.
func (sm *SelfMonitoring) SetName(name string) {
	sm.name = name
}

// GetName returns the base name for the self-monitoring collector's Kubernetes resources.
func (sm *SelfMonitoring) GetName() string {
	return sm.name + NameSuffix
}

// IsEnabled reports whether the self-monitoring collector should be deployed.
func (sm *SelfMonitoring) IsEnabled() bool {
	return sm.Enabled
}
