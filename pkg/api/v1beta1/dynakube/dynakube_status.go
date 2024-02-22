package dynakube

import (
	"fmt"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct { //nolint:revive

	// Observed state of OneAgent
	OneAgent OneAgentStatus `json:"oneAgent,omitempty"`

	// Observed state of ActiveGate
	ActiveGate ActiveGateStatus `json:"activeGate,omitempty"`

	// Observed state of Code Modules
	CodeModules CodeModulesStatus `json:"codeModules,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Observed state of Dynatrace API
	DynatraceApi DynatraceApiStatus `json:"dynatraceApi,omitempty"`

	// LastTokenProbeTimestamp tracks when the last request for the API token validity was sent
	// +kubebuilder:deprecatedversion:warning="Use DynatraceApiStatus.LastTokenScopeRequest instead"
	LastTokenProbeTimestamp *metav1.Time `json:"lastTokenProbeTimestamp,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type DynatraceApiStatus struct {
	// Time of the last token request
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

	// Time of the last connection request
	LastRequest metav1.Time `json:"lastRequest,omitempty"`
	// UUID of the tenant, received from the tenant
	TenantUUID string `json:"tenantUUID,omitempty"`

	// Available connection endpoints
	Endpoints string `json:"endpoints,omitempty"`
}

type OneAgentConnectionInfoStatus struct {
	// Information for communicating with the tenant
	ConnectionInfoStatus `json:",inline"`

	// List of communication hosts
	CommunicationHosts []CommunicationHostStatus `json:"communicationHosts,omitempty"`
}

type ActiveGateConnectionInfoStatus struct {
	// Information about Active Gate's connections
	ConnectionInfoStatus `json:",inline"`
}

type CommunicationHostStatus struct {
	// Connection protocol
	Protocol string `json:"protocol,omitempty"`

	// Host domain
	Host string `json:"host,omitempty"`

	// Connection port
	Port uint32 `json:"port,omitempty"`
}

type ActiveGateStatus struct {
	status.VersionStatus `json:",inline"`

	// Information about Active Gate's connections
	ConnectionInfoStatus ActiveGateConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

type CodeModulesStatus struct {
	status.VersionStatus `json:",inline"`
}

type OneAgentStatus struct {
	status.VersionStatus `json:",inline"`

	// List of deployed OneAgent instances
	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	// Time of the last instance status update
	LastInstanceStatusUpdate *metav1.Time `json:"lastInstanceStatusUpdate,omitempty"`

	// Time of the last process module config update
	LastProcessModuleConfigUpdate *metav1.Time `json:"lastProcessModuleConfigUpdate,omitempty"`

	// Commands used for OneAgent's readiness probe
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Healthcheck *containerv1.HealthConfig `json:"healthcheck,omitempty"`

	// Information about OneAgent's connections
	ConnectionInfoStatus OneAgentConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

type OneAgentInstance struct {
	// Name of the OneAgent pod
	PodName string `json:"podName,omitempty"`

	// IP address of the pod
	IPAddress string `json:"ipAddress,omitempty"`
}

// SetPhase sets the status phase on the DynaKube object.
func (dk *DynaKubeStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.Phase
	dk.Phase = phase

	return upd
}
