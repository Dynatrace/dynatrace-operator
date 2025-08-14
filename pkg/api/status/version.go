// +kubebuilder:object:generate=true
// +k8s:openapi-gen=true
package status

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type VersionSource string

const (
	TenantRegistryVersionSource VersionSource = "tenant-registry"
	CustomImageVersionSource    VersionSource = "custom-image"
	CustomVersionVersionSource  VersionSource = "custom-version"
	PublicRegistryVersionSource VersionSource = "public-registry"

	ImmutableImageType = "immutable"
)

type VersionStatus struct {
	// Indicates when the last check for a new version was performed
	LastProbeTimestamp *metav1.Time `json:"lastProbeTimestamp,omitempty"`
	// Source of the image (tenant-registry, public-registry, ...)
	Source VersionSource `json:"source,omitempty"`
	// Image ID
	ImageID string `json:"imageID,omitempty"`
	// Image version
	Version string `json:"version,omitempty"`
	// Image type
	Type string `json:"type,omitempty"`
}

// IsZero returns true if the VersionStatus fields are not initialized.
func (status *VersionStatus) IsZero() bool {
	return status == nil || *status == VersionStatus{}
}
