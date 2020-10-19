package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DynaKubeSpec defines the desired state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	BaseDynaKubeSpec `json:""`

	// Enabled capabilities of the DynaKube. E.g.: "kubernetes_monitoring"
	// +kubebuilder:validation:Required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="DynaKube Capabilities"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Capabilities []string `json:"dt_capabilities"`

	NetworkZone string `json:"networkZone,omitempty"`

	// Disable automatic restarts of Activegate pods in case a new version is available
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Disable Activegate update"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	DisableActivegateUpdate bool `json:"disableActivegateUpdate,omitempty"`

	// The public API endpoint of the Kubernetes instance
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Kubernetes API Endpoint"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	KubernetesAPIEndpoint string `json:"kubernetesApiEndpoint,omitempty"`

	// Amount of replicas for your DynaKube
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Replicas"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:number"
	Replicas *int32 `json:"replicas,omitempty"`
}

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions status.Conditions `json:"conditions,omitempty"`

	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`

	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase DynaKubePhaseType `json:"phase,omitempty"`
}

type DynaKubePhaseType string

type DynaKubeInstance struct {
	PodName   string `json:"podName,omitempty"`
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

const (
//Commented for linter, uncomment if needed
//Running   DynaKubePhaseType = "Running"
//Deploying DynaKubePhaseType = "Deploying"
//Error     DynaKubePhaseType = "Error"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dynakubes,scope=Namespaced
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.spec.tokens`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Dynatrace DynaKube"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Pod,v1,""`
type DynaKube struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynaKubeSpec   `json:"spec,omitempty"`
	Status DynaKubeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKubeList contains a list of DynaKube
type DynaKubeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynaKube `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynaKube{}, &DynaKubeList{})
}

//Commented for linter, uncomment if needed
//const (
//	// APITokenConditionType identifies the API Token validity condition
//	APITokenConditionType status.ConditionType = "APIToken"
//
//	// PaaSTokenConditionType identifies the PaaS Token validity condition
//	PaaSTokenConditionType status.ConditionType = "PaaSToken"
//)
//
//// Possible reasons for ApiToken and PaaSToken conditions
//const (
//	// ReasonTokenReady is set when a token has passed verifications
//	ReasonTokenReady status.ConditionReason = "TokenReady"
//
//	// ReasonTokenSecretNotFound is set when the referenced secret can't be found
//	ReasonTokenSecretNotFound status.ConditionReason = "TokenSecretNotFound"
//
//	// ReasonTokenMissing is set when the field is missing on the secret
//	ReasonTokenMissing status.ConditionReason = "TokenMissing"
//
//	// ReasonTokenUnauthorized is set when a token is unauthorized to query the Dynatrace API
//	ReasonTokenUnauthorized status.ConditionReason = "TokenUnauthorized"
//
//	// ReasonTokenScopeMissing is set when the token is missing the required scope for the Dynatrace API
//	ReasonTokenScopeMissing status.ConditionReason = "TokenScopeMissing"
//
//	// ReasonTokenError is set when an unknown error has been found when verifying the token
//	ReasonTokenError status.ConditionReason = "TokenError"
//)
