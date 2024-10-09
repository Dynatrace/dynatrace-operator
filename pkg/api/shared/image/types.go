package image

// +kubebuilder:object:generate=true
type Ref struct {
	// Custom image repository
	// +kubebuilder:example:="docker.io/dynatrace/image-name"
	Repository string `json:"repository,omitempty"`

	// Indicates a tag of the image to use
	Tag string `json:"tag,omitempty"`
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
