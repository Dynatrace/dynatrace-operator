package tenant

// +kubebuilder:object:generate=true

type Status struct {
	Phase int `json:"phase,omitempty"`
}
