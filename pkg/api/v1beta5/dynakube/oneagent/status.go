package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true

type Status struct {
	status.VersionStatus `json:",inline"`

	// List of deployed OneAgent instances
	Instances map[string]Instance `json:"instances,omitempty"`

	// Time of the last instance status update
	LastInstanceStatusUpdate *metav1.Time `json:"lastInstanceStatusUpdate,omitempty"`

	// Commands used for OneAgent's readiness probe
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Healthcheck *containerv1.HealthConfig `json:"healthcheck,omitempty"`

	// Information about OneAgent's connections
	ConnectionInfoStatus ConnectionInfoStatus `json:"connectionInfoStatus,omitempty"`
}

// +kubebuilder:object:generate=true

type Instance struct {
	// Name of the OneAgent pod
	PodName string `json:"podName,omitempty"`

	// IP address of the pod
	IPAddress string `json:"ipAddress,omitempty"`
}

// +kubebuilder:object:generate=true

type ConnectionInfoStatus struct {
	// Information for communicating with the tenant
	communication.ConnectionInfo `json:",inline"`

	// List of communication hosts
	CommunicationHosts []CommunicationHostStatus `json:"communicationHosts,omitempty"`
}

// +kubebuilder:object:generate=true

type CommunicationHostStatus struct {
	// Connection protocol
	Protocol string `json:"protocol,omitempty"`

	// Host domain
	Host string `json:"host,omitempty"`

	// Connection port
	Port uint32 `json:"port,omitempty"`
}
