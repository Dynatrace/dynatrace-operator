// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package metadataenrichment

// +kubebuilder:object:generate=true

type Status struct {
	Rules []Rule `json:"rules,omitempty"`
}
