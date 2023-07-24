// +kubebuilder:object:generate=true
// +k8s:openapi-gen=true
package status

type PhaseType string

const (
	Running   PhaseType = "Running"
	Deploying PhaseType = "Deploying"
	Error     PhaseType = "Error"
)
