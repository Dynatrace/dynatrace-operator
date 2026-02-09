package image

import corev1 "k8s.io/api/core/v1"

// +kubebuilder:object:generate=true

type Ref struct {
	// Custom image repository
	// +kubebuilder:example:="docker.io/dynatrace/image-name"
	Repository string `json:"repository,omitempty"`

	// Indicates a tag of the image to use
	Tag string `json:"tag,omitempty"`

	// Image pull policy to use
	PullPolicy PullPolicy `json:"pullPolicy,omitempty"`
}

// StringWithDefaults will use the provided default values for fields that were not already set.
func (ref Ref) StringWithDefaults(repo, tag string) string {
	if ref.Repository == "" {
		ref.Repository = repo
	}

	if ref.Tag == "" {
		ref.Tag = tag
	}

	return ref.String()
}

func (ref Ref) String() string {
	return ref.Repository + ":" + ref.Tag
}

// IsZero returns true if the image ref is empty.
// Prefer this name over IsEmpty for compatibility with JSON omitzero.
func (ref *Ref) IsZero() bool {
	return ref == nil || *ref == Ref{}
}

// GetPolicy returns the image pull policy.
func (ref Ref) GetPullPolicy() corev1.PullPolicy {
	return corev1.PullPolicy(ref.PullPolicy)
}

// +kubebuilder:validation:Enum=IfNotPresent;Always;Never

// PullPolicy is the image pull policy. Use a custom type to share the validation marker.
type PullPolicy string
