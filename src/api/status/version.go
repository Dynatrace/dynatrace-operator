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
)

type VersionStatus struct {
	Source             VersionSource `json:"source,omitempty"`
	ImageID            string        `json:"imageID,omitempty"`
	Version            string        `json:"version,omitempty"`
	LastProbeTimestamp *metav1.Time  `json:"lastProbeTimestamp,omitempty"`
}
