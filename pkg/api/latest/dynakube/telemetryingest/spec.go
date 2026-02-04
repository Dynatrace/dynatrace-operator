package telemetryingest

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

	// +listType=set
	// +kubebuilder:validation:Optional
	Protocols []string `json:"protocols,omitempty"`
}
