// +kubebuilder:object:generate=true
// +k8s:openapi-gen=true
package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type VersionSource string

const (
	TenantRegistryVersionSource VersionSource = "tenant-registry"
	CustomImageVersionSource    VersionSource = "custom-image"
	CustomVersionVersionSource  VersionSource = "custom-version"
	PublicRegistryVersionSource VersionSource = "public-registry"

	ImmutableImageType = "immutable"
)

type VersionStatus struct {
	// Source of the image (tenant-registry, public-registry, ...)
	Source VersionSource `json:"source,omitempty"`
	// Image ID
	ImageID string `json:"imageID,omitempty"`
	// Image version
	Version string `json:"version,omitempty"`
	// Image type OnAgent Image type (immutable, mutable)
	Type string `json:"type,omitempty"`
	// Indicates when the last check for a new version was performed
	LastProbeTimestamp *metav1.Time `json:"lastProbeTimestamp,omitempty"`
}

func (s *VersionStatus) CustomImageNeedsReconciliation(logImageNeedsReconciliationMessage LogFn, customImage string) bool {
	if s.Source == CustomImageVersionSource {
		oldImage := s.ImageID
		newImage := customImage
		// The old image is can be the same as the new image (if only digest was given, or a tag was given but couldn't get the digest)
		// or the old image is the same as the new image but with the digest added to the end of it (if a tag was provide, and we could append the digest to the end)
		// or the 2 images are different
		if !strings.HasPrefix(oldImage, newImage) {
			logImageNeedsReconciliationMessage()
			return true
		}
	}
	return false
}

type LogFn func()

func (s *VersionStatus) CustomVersionNeedsReconciliation(logVersionNeedsReconciliationMessage LogFn, customVersion string) bool {
	if s.Source == CustomVersionVersionSource {
		oldVersion := s.Version
		newVersion := customVersion
		if oldVersion != newVersion {
			logVersionNeedsReconciliationMessage()
			return true
		}
	}
	return false
}
