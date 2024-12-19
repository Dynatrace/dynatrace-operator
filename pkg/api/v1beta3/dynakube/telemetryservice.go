package dynakube

type TelemetryServiceSpec struct {
	// +kubebuilder:validation:Optional
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	TlsRefName string `json:"tlsRefName,omitempty"`

	// +kubebuilder:validation:Optional
	Protocols []string `json:"protocols,omitempty"`
}
