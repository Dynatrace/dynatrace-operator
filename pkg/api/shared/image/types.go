// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package image

import corev1 "k8s.io/api/core/v1"

// +kubebuilder:object:generate=true

type Ref struct {
	// Custom image repository
	// +kubebuilder:example:="docker.io/dynatrace/image-name"
	Repository string `json:"repository,omitempty"`

	// Indicates a tag of the image to use
	Tag string `json:"tag,omitempty"`

	// Digest pins the image to a specific content-addressable hash in the OCI
	// `<algorithm>:<hex>` form (e.g. `sha256:abc...`). When set, the tag is ignored
	// when rendering the image reference.
	// +kubebuilder:validation:Pattern:="^[a-z0-9]+:([a-f0-9]+|[A-F0-9]+)$"
	Digest string `json:"digest,omitempty"`

	// Image pull policy to use
	// +kubebuilder:validation:Enum=IfNotPresent;Always;Never
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// StringWithDefaults will use the provided default values for fields that were not already set.
// If a digest is present (either on the ref or via the default), the tag is omitted from the
// rendered image reference.
func (ref Ref) StringWithDefaults(repo, tag string) string {
	if ref.Repository == "" {
		ref.Repository = repo
	}

	if ref.Tag == "" {
		ref.Tag = tag
	}

	return ref.String()
}

// String renders the image reference. If a digest is set, the tag is omitted to avoid the
// confusing case where the tag and digest disagree — the digest always wins.
// Returns the empty string if repository and tag/digest are not set.
func (ref Ref) String() string {
	if ref.Repository != "" && (ref.Tag != "" || ref.Digest != "") {
		if ref.Digest != "" {
			return ref.Repository + "@" + ref.Digest
		}

		return ref.Repository + ":" + ref.Tag
	}

	return ""
}
