package metadataenrichment

// +kubebuilder:object:generate=true

type Status struct {
	Rules []Rule `json:"rules,omitempty"`
}
