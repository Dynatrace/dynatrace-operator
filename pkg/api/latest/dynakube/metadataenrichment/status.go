package metadataenrichment

// +kubebuilder:object:generate=true
type Status struct {
	Rules []EnrichmentRule `json:"rules,omitempty"`
}
