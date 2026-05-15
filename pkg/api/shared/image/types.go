package image

import corev1 "k8s.io/api/core/v1"

// +kubebuilder:object:generate=true

type Ref struct {
	// Custom image repository
	// +kubebuilder:example:="docker.io/dynatrace/image-name"
	Repository string `json:"repository,omitempty"`

	// Indicates a tag of the image to use
	Tag string `json:"tag,omitempty"`

	// Digest pins the image to a specific content-addressable hash (e.g. sha256:...).
	// When set, the tag is ignored when rendering the image reference.
	Digest string `json:"digest,omitempty"`

	// Image pull policy to use
	PullPolicy PullPolicy `json:"pullPolicy,omitempty"`
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
func (ref Ref) String() string {
	if ref.Digest != "" {
		return ref.Repository + "@" + ref.Digest
	}

	return ref.Repository + ":" + ref.Tag
}

// HasImage returns true when the ref points to a resolvable image — i.e. a repository
// plus at least one of a tag or a digest.
func (ref *Ref) HasImage() bool {
	if ref == nil {
		return false
	}

	return ref.Repository != "" && (ref.Tag != "" || ref.Digest != "")
}

// GetPolicy returns the image pull policy.
func (ref Ref) GetPullPolicy() corev1.PullPolicy {
	return corev1.PullPolicy(ref.PullPolicy)
}

// +kubebuilder:validation:Enum=IfNotPresent;Always;Never

// PullPolicy is the image pull policy. Use a custom type to share the validation marker.
type PullPolicy string
