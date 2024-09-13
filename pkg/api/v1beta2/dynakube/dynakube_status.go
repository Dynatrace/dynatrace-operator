package dynakube

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Observed state of Metadata-Enrichment
	MetadataEnrichment MetadataEnrichmentStatus `json:"metadataEnrichment,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Observed state of Dynatrace API
	DynatraceApi DynatraceApiStatus `json:"dynatraceApi,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`

	// KubernetesClusterMEID contains the ID of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterMEID string `json:"kubernetesClusterMEID,omitempty"`

	// KubernetesClusterName contains the display name (also know as label) of the monitored entity that points to the Kubernetes cluster
	KubernetesClusterName string `json:"kubernetesClusterName,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type DynatraceApiStatus struct {
	// Time of the last token request
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`
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

	// The ClusterIPs set by Kubernetes on the ActiveGate Service created by the Operator
	ServiceIPs []string `json:"serviceIPs,omitempty"`
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

type EnrichmentRuleType string

const (
	EnrichmentLabelRule      EnrichmentRuleType = "LABEL"
	EnrichmentAnnotationRule EnrichmentRuleType = "ANNOTATION"
)

const MetadataPrefix string = "metadata.dynatrace.com"

type MetadataEnrichmentStatus struct {
	Rules []EnrichmentRule `json:"rules,omitempty"`
}

type EnrichmentRule struct {
	Type    EnrichmentRuleType `json:"type,omitempty"`
	Source  string             `json:"source,omitempty"`
	Target  string             `json:"target,omitempty"`
	Enabled bool               `json:"enabled,omitempty"`
}

func (rule EnrichmentRule) ToAnnotationKey() string {
	return MetadataPrefix + "/" + rule.Target
}

// SetPhase sets the status phase on the DynaKube object.
func (dk *DynaKubeStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != dk.Phase
	dk.Phase = phase

	return upd
}

func (dk *DynaKube) UpdateStatus(ctx context.Context, client client.Client) error {
	dk.Status.UpdatedTimestamp = metav1.Now()
	err := client.Status().Update(ctx, dk)

	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update dynakube due to conflict", "name", dk.Name)

		return nil
	}

	return errors.WithStack(err)
}
