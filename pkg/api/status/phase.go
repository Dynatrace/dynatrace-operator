// +kubebuilder:object:generate=true
// +k8s:openapi-gen=true
package status

type DeploymentPhase string

const (
	Running   DeploymentPhase = "Running"
	Deploying DeploymentPhase = "Deploying"
	Error     DeploymentPhase = "Error"
)
