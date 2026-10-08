// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package extensions

type Status struct {
	// The resolved EEC image that is currently deployed.
	// The JSON tag uses "image" (matching the Spec field name) while the Go field name is ResolvedImage to distinguish it from Spec.Image.
	ResolvedImage string `json:"image,omitempty"`
}
