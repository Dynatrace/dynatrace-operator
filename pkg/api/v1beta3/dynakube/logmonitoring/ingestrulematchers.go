package logmonitoring

// +kubebuilder:object:generate=true

type IngestRuleMatchers struct {
	// +kubebuilder:validation:Optional
	Attribute string `json:"attribute,omitempty"`

	// +kubebuilder:validation:Optional
	Values []string `json:"values,omitempty"`
}
