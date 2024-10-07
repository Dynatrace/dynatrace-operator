package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
type ConnectionInfo struct {

	// Time of the last connection request
	LastRequest metav1.Time `json:"lastRequest,omitempty"`
	// UUID of the tenant, received from the tenant
	TenantUUID string `json:"tenantUUID,omitempty"`

	// Available connection endpoints
	Endpoints string `json:"endpoints,omitempty"`
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
