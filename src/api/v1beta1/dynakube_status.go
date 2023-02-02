package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct {
	// Defines the current state (Running, Updating, Error, ...)
	Phase DynaKubePhaseType `json:"phase,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Deprecated: use DynatraceApiStatus.LastTokenScopeRequest instead
	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`

	// Deprecated: use LastAPITokenProbeTimestamp instead
	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`

	// Deprecated: use LastAPITokenProbeTimestamp instead
	// LastDataIngestTokenProbeTimestamp tracks when the last request for the DataIngest token validity was sent
	LastDataIngestTokenProbeTimestamp *metav1.Time `json:"lastDataIngestTokenProbeTimestamp,omitempty"`

	// Credentials used to connect back to Dynatrace.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API and PaaS Tokens"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	Tokens string `json:"tokens,omitempty"`

	// LastClusterVersionProbeTimestamp indicates when the cluster's version was last checked
	LastClusterVersionProbeTimestamp *metav1.Time `json:"lastClusterVersionProbeTimestamp,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// ConnectionInfo caches information about the tenant and its communication hosts
	ConnectionInfo ConnectionInfoStatus `json:"connectionInfo,omitempty"`

	// CommunicationHostForClient caches a communication host specific to the api url.
	CommunicationHostForClient CommunicationHostStatus `json:"communicationHostForClient,omitempty"`

	// LatestAgentVersionUnixDefault caches the current agent version for unix and the default installer which is configured for the environment
	LatestAgentVersionUnixDefault string `json:"latestAgentVersionUnixDefault,omitempty"`

	// LatestAgentVersionUnixDefault caches the current agent version for unix and the PaaS installer which is configured for the environment
	LatestAgentVersionUnixPaas string `json:"latestAgentVersionUnixPaas,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ActiveGate   ActiveGateStatus   `json:"activeGate,omitempty"`
	OneAgent     OneAgentStatus     `json:"oneAgent,omitempty"`
	Synthetic    SyntheticStatus    `json:"synthetic,omitempty"`
	DynatraceApi DynatraceApiStatus `json:"dynatraceApi,omitempty"`
}

const MaxRequestInterval = 15 * time.Minute

type DynatraceApiStatus struct {
	LastTokenScopeRequest               metav1.Time `json:"LastTokenScopeRequest,omitempty"`
	LastOneAgentConnectionInfoRequest   metav1.Time `json:"LastOneAgentConnectionInfoRequest,omitempty"`
	LastActiveGateConnectionInfoRequest metav1.Time `json:"LastActiveGateConnectionInfoRequest,omitempty"`
}

func (dynatraceApiStatus *DynatraceApiStatus) ResetCachedTimestamps() {
	dynatraceApiStatus.LastTokenScopeRequest = metav1.Time{}
	dynatraceApiStatus.LastOneAgentConnectionInfoRequest = metav1.Time{}
	dynatraceApiStatus.LastActiveGateConnectionInfoRequest = metav1.Time{}
}

func CacheValidMessage(functionName string) string {
	return fmt.Sprintf("skipping %s, last request was made less than %d minutes ago",
		functionName,
		int(MaxRequestInterval.Minutes()))
}

func IsRequestOutdated(time metav1.Time) bool {
	return time.Add(MaxRequestInterval).Before(metav1.Now().Time)
}

type ConnectionInfoStatus struct {
	CommunicationHosts              []CommunicationHostStatus `json:"communicationHosts,omitempty"`
	TenantUUID                      string                    `json:"tenantUUID,omitempty"`
	FormattedCommunicationEndpoints string                    `json:"formattedCommunicationEndpoints,omitempty"`
}

type CommunicationHostStatus struct {
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     uint32 `json:"port,omitempty"`
}

// +kubebuilder:object:generate=false
type VersionStatusNamer interface {
	Status() VersionStatus
	Name() string
}

type VersionStatus struct {
	// ImageHash contains the last image hash seen.
	ImageHash string `json:"imageHash,omitempty"`

	// Version contains the version to be deployed.
	Version string `json:"version,omitempty"`

	// LastUpdateProbeTimestamp defines the last timestamp when the querying for updates have been done
	LastUpdateProbeTimestamp *metav1.Time `json:"lastUpdateProbeTimestamp,omitempty"`
}

func (verStatus *VersionStatus) Status() VersionStatus {
	return *verStatus.DeepCopy()
}

var _ VersionStatusNamer = (*ActiveGateStatus)(nil)

type ActiveGateStatus struct {
	VersionStatus `json:",inline"`
}

func (agStatus *ActiveGateStatus) Name() string {
	return "ActiveGate"
}

var _ VersionStatusNamer = (*OneAgentStatus)(nil)

type OneAgentStatus struct {
	VersionStatus `json:",inline"`

	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	LastInstanceStatusUpdate *metav1.Time `json:"LastInstanceStatusUpdate,omitempty"`
}

func (oneAgentStatus *OneAgentStatus) Name() string {
	return "OneAgent"
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

var _ VersionStatusNamer = (*SyntheticStatus)(nil)

type SyntheticStatus struct {
	VersionStatus `json:",inline"`
}

func (syn *SyntheticStatus) Name() string {
	return "Synthetic"
}

type DynaKubePhaseType string

const (
	Running   DynaKubePhaseType = "Running"
	Deploying DynaKubePhaseType = "Deploying"
	Error     DynaKubePhaseType = "Error"
)

// SetPhase sets the status phase on the DynaKube object
func (dk *DynaKubeStatus) SetPhase(phase DynaKubePhaseType) bool {
	upd := phase != dk.Phase
	dk.Phase = phase
	return upd
}

// SetPhaseOnError fills the phase with the Error value in case of any error
func (dk *DynaKubeStatus) SetPhaseOnError(err error) bool {
	if err != nil {
		return dk.SetPhase(Error)
	}
	return false
}
