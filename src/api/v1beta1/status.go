package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetCacheValidMessage(functionName string, lastRequestTimestamp metav1.Time, timeout time.Duration) string {
	remaining := timeout - time.Since(lastRequestTimestamp.Time)
	return fmt.Sprintf("skipping %s, last request was made less than %d minutes ago, %d minutes remaining until next request",
		functionName,
		int(timeout.Minutes()),
		int(remaining.Minutes()))
}

type ConnectionInfoStatus struct {
	TenantUUID  string      `json:"tenantUUID,omitempty"`
	Endpoints   string      `json:"endpoints,omitempty"`
	LastRequest metav1.Time `json:"lastRequest,omitempty"`
}

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
