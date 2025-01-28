package telemetryservice

type TelemetryService struct {
	*Spec

	name string
}

// +kubebuilder:object:generate=true

type Spec struct {
	// +kubebuilder:validation:Optional
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	TlsRefName string `json:"tlsRefName,omitempty"`

	// +kubebuilder:validation:Optional
	Protocols []string `json:"protocols,omitempty"`
}
