package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TokenConditionType identifies the token validity condition
	TokenConditionType string = "Tokens"

	// APITokenConditionType identifies the API Token validity condition
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition
	PaaSTokenConditionType string = "PaaSToken"

	// DataIngestTokenConditionType identifies the DataIngest Token validity condition
	DataIngestTokenConditionType string = "DataIngestToken"
)

// Possible reasons for ApiToken and PaaSToken conditions
const (
	// ReasonTokenReady is set when a token has passed verifications
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenError is set when an unknown error has been found when verifying the token
	ReasonTokenError string = "TokenError"
)

type DynaKubeProxy struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

type DynaKubeValueSource struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:storageversion

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dynakubes,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:csv:customresourcedefinitions:displayName="Dynatrace DynaKube"
// +operator-sdk:csv:customresourcedefinitions:resources={{StatefulSet,v1,},{DaemonSet,v1,},{Pod,v1,}}
type DynaKube struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynaKubeSpec   `json:"spec,omitempty"`
	Status DynaKubeStatus `json:"status,omitempty"`
}

// DynaKubeSpec defines the desired state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeSpec struct {
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
	Proxy *DynaKubeProxy `json:"proxy,omitempty"`

	// Optional: Adds custom RootCAs from a configmap
	// This property only affects certificates used to communicate with the Dynatrace API.
	// The property is not applied to the ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trusted CAs",order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ConfigMap"}
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Optional: Sets Network Zone for OneAgent and ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`

	// If enabled, Istio on the cluster will be configured automatically to allow access to the Dynatrace environment
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Istio automatic management",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	EnableIstio bool `json:"enableIstio,omitempty"`

	// Optional: set a namespace selector to limit which namespaces are monitored
	// By default, all namespaces will be monitored
	// Has no effect during classicFullStack and hostMonitoring mode
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// General configuration about OneAgent instances
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	OneAgent OneAgentSpec `json:"oneAgent,omitempty"`

	// General configuration about ActiveGate instances
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ActiveGate",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ActiveGate ActiveGateSpec `json:"activeGate,omitempty"`

	//  Deprecated: Configuration for Routing
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Routing"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Routing RoutingSpec `json:"routing,omitempty"`

	//  Deprecated: Configuration for Kubernetes Monitoring
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Kubernetes Monitoring"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	KubernetesMonitoring KubernetesMonitoringSpec `json:"kubernetesMonitoring,omitempty"`
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
