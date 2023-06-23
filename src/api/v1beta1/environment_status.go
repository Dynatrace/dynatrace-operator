package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


type DynatraceApiStatus struct {
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`
}

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

type OneAgentConnectionInfoStatus struct {
	ConnectionInfoStatus `json:",inline"`
	CommunicationHosts   []CommunicationHostStatus `json:"communicationHosts,omitempty"`
}

type ActiveGateConnectionInfoStatus struct {
	ConnectionInfoStatus `json:",inline"`
}

type CommunicationHostStatus struct {
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     uint32 `json:"port,omitempty"`
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

type CodeModulesStatus struct {
	VersionStatus `json:",inline"`
}

type OneAgentStatus struct {
	VersionStatus `json:",inline"`

	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	LastInstanceStatusUpdate *metav1.Time `json:"lastInstanceStatusUpdate,omitempty"`

	ConnectionInfoStatus OneAgentConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

type SyntheticStatus struct {
	VersionStatus `json:",inline"`
}
