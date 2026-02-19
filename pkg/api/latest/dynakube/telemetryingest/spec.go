package telemetryingest

import "github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"

type TelemetryIngest struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

type Spec struct {
	// +kubebuilder:validation:Optional
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	TLSRefName string `json:"tlsRefName,omitempty"`

	// +kubebuilder:validation:Optional
	Protocols []otelcgen.Protocol `json:"protocols,omitempty"`
}
