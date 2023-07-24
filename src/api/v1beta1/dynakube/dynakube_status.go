package dynakube

import (
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct { // nolint:revive
	// Defines the current state (Running, Updating, Error, ...)
	Phase status.PhaseType `json:"phase,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Deprecated: use DynatraceApiStatus.LastTokenScopeRequest instead
	// LastTokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastTokenProbeTimestamp *metav1.Time `json:"lastTokenProbeTimestamp,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ActiveGate   ActiveGateStatus   `json:"activeGate,omitempty"`
	OneAgent     OneAgentStatus     `json:"oneAgent,omitempty"`
	CodeModules  CodeModulesStatus  `json:"codeModules,omitempty"`
	Synthetic    SyntheticStatus    `json:"synthetic,omitempty"`
	DynatraceApi DynatraceApiStatus `json:"dynatraceApi,omitempty"`
}

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

type ActiveGateStatus struct {
	status.VersionStatus `json:",inline"`
	ConnectionInfoStatus ActiveGateConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

type CodeModulesStatus struct {
	status.VersionStatus `json:",inline"`
}

type OneAgentStatus struct {
	status.VersionStatus `json:",inline"`

	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	LastInstanceStatusUpdate *metav1.Time `json:"lastInstanceStatusUpdate,omitempty"`

	ConnectionInfoStatus OneAgentConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

type SyntheticStatus struct {
	status.VersionStatus `json:",inline"`
}

// SetPhase sets the status phase on the DynaKube object
func (dk *DynaKubeStatus) SetPhase(phase status.PhaseType) bool {
	upd := phase != dk.Phase
	dk.Phase = phase
	return upd
}
