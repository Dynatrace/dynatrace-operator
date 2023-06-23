package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TokenConditionType string = "Tokens"

	APITokenConditionType string = "APIToken"

	PaaSTokenConditionType string = "PaaSToken"

	DataIngestTokenConditionType string = "DataIngestToken"
)

const (
	ReasonTokenReady string = "TokenReady"
	ReasonTokenError string = "TokenError"
)

type StringValueSource struct {
	Value string `json:"value,omitempty"`

	ValueFrom string `json:"valueFrom,omitempty"`
}

type EnvironmentValueSource struct {
	Value *Environment `json:"value,omitempty"`

	ValueFrom string `json:"valueFrom,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec      `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

type EnvironmentSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Location of the Dynatrace API to connect to, including your specific environment UUID
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API URL",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Credentials for the DynaKube to connect back to Dynatrace.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant specific secrets",order=2,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Optional: Pull secret for your private registry
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom PullSecret",order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Disable certificate validation checks for installer download and API communication
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// Optional: Set custom proxy settings either directly or from a secret with the field 'proxy'
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Proxy *StringValueSource `json:"proxy,omitempty"`

	// Optional: Adds custom RootCAs from a configmap
	// This property only affects certificates used to communicate with the Dynatrace API.
	// The property is not applied to the ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trusted CAs",order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ConfigMap"}
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Optional: Sets Network Zone for OneAgent and ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object


type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

// EnvironmentStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type EnvironmentStatus struct {
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Deprecated: use DynatraceApiStatus.LastTokenScopeRequest instead
	// LastTokenProbeTimestamp tracks when the last request for the API token validity was sent
	ApiStatus DynatraceApiStatus `json:"apiStatus,omitempty"`

	// KubeSystemUUID contains the UUID of the current Kubernetes cluster
	KubeSystemUUID string `json:"kubeSystemUUID,omitempty"`
}

type DynatraceApiStatus struct {
	LastTokenScopeRequest metav1.Time `json:"lastTokenScopeRequest,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
	SchemeBuilder.Register(&ActiveGate{}, &ActiveGateList{})
	SchemeBuilder.Register(&OneAgent{}, &OneAgentList{})
}
