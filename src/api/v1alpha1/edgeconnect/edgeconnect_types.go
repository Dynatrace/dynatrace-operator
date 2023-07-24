package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeConnectSpec defines the desired state of EdgeConnect
// +k8s:openapi-gen=true
type EdgeConnectSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Location of the Dynatrace API to connect to, including your specific environment UUID
	// +kubebuilder:validation:Required
	ApiServer string `json:"apiServer"`

	// Authorization configuration
	Oauth OAuthSpec `json:"oauth,omitempty"`

	HostRestrictions string `json:"hostRestrictions"`

	// Image reference
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// AutoUpdate
	AutoUpdate bool `json:"autoUpdate,omitempty"`

	// Optional: Pull secret for your private registry
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Optional: Adds additional annotations for the EdgeConnect pods
	Annotations map[string]string `json:"annotations,omitempty"`

	// Optional: Adds additional labels for the EdgeConnect pods
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: List of environment variables to set for the EdgeConnect
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Amount of replicas for your DynaKube
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: define resources requests and limits for single pods
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Node selector to control the selection of nodes (optional)
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the EdgeConnect pods
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: set topology spread constraints for the EdgeConnect pods
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

type OAuthSpec struct {
	// Credentials for the EdgeConnect to connect back to Dynatrace.
	ClientSecret string `json:"clientSecret,omitempty"`
	// Optional: the ActiveGate container image. Defaults to the latest ActiveGate image provided by the Docker Registry
	// implementation from the Dynatrace environment set as API URL.
	Endpoint string `json:"endpoint,omitempty"`
	// Optional: the ActiveGate container image. Defaults to the latest ActiveGate image provided by the Docker Registry
	// implementation from the Dynatrace environment set as API URL.
	Resource string `json:"resource,omitempty"`
}

type ImageRefSpec struct {
	// Optional: If specified, indicates the EdgeConnect repository to use
	Repository string `json:"repository,omitempty"`

	// Optional: tag
	Tag string `json:"tag,omitempty"`
}

// EdgeConnectStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type EdgeConnectStatus struct {
	// Defines the current state (Running, Updating, Error, ...)
	Phase   EdgeConnectPhaseType `json:"phase,omitempty"`
	Version VersionStatus        `json:"version,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type VersionSource string

const (
	TenantRegistryVersionSource VersionSource = "tenant-registry"
	CustomImageVersionSource    VersionSource = "custom-image"
	CustomVersionVersionSource  VersionSource = "custom-version"
	PublicRegistryVersionSource VersionSource = "public-registry"
)

type VersionStatus struct {
	Source             VersionSource `json:"source,omitempty"`
	ImageID            string        `json:"imageID,omitempty"`
	Version            string        `json:"version,omitempty"`
	LastProbeTimestamp *metav1.Time  `json:"lastProbeTimestamp,omitempty"`
}

type EdgeConnectPhaseType string

const (
	Running   EdgeConnectPhaseType = "Running"
	Deploying EdgeConnectPhaseType = "Deploying"
	Error     EdgeConnectPhaseType = "Error"
)

// SetPhase sets the status phase on the DynaKube object
func (dk *EdgeConnectStatus) SetPhase(phase EdgeConnectPhaseType) bool {
	upd := phase != dk.Phase
	dk.Phase = phase
	return upd
}

// SetPhaseOnError fills the phase with the Error value in case of any error
func (dk *EdgeConnectStatus) SetPhaseOnError(err error) bool {
	if err != nil {
		return dk.SetPhase(Error)
	}
	return false
}

const (
	// APITokenConditionType identifies the API Token validity condition
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition
	PaaSTokenConditionType string = "PaaSToken"
)

// Possible reasons for ApiToken and PaaSToken conditions
const (
	// ReasonTokenReady is set when a token has passed verifications
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenSecretNotFound is set when the referenced secret can't be found
	ReasonTokenSecretNotFound string = "TokenSecretNotFound"

	// ReasonTokenMissing is set when the field is missing on the secret
	ReasonTokenMissing string = "TokenMissing"

	// ReasonTokenUnauthorized is set when a token is unauthorized to query the Dynatrace API
	ReasonTokenUnauthorized string = "TokenUnauthorized"

	// ReasonTokenScopeMissing is set when the token is missing the required scope for the Dynatrace API
	ReasonTokenScopeMissing string = "TokenScopeMissing"

	// ReasonTokenError is set when an unknown error has been found when verifying the token
	ReasonTokenError string = "TokenError"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnect is the Schema for the EdgeConnect API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=edgeconnets,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiServer`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.status.tokens`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:csv:customresourcedefinitions:displayName="Dynatrace EdgeConnect"
// +operator-sdk:csv:customresourcedefinitions:resources={{StatefulSet,v1,},{DaemonSet,v1,},{Pod,v1,}}
// +kubebuilder:storageversion
type EdgeConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeConnectSpec   `json:"spec,omitempty"`
	Status EdgeConnectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeConnectList contains a list of DynaKube
type EdgeConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeConnect `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeConnect{}, &EdgeConnectList{})
}
