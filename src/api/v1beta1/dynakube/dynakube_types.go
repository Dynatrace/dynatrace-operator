// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1beta1
package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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

type DynaKubeProxy struct { // nolint:revive
	// Proxy URL. It has preference over ValueFrom.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// Secret containing proxy URL.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

type DynaKubeValueSource struct { // nolint:revive
	// Custom properties value.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// Custom properties secret.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
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
type DynaKubeSpec struct { // nolint:revive
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address.
	// For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id).
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API URL",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Name of the secret holding the tokens used for connecting to Dynatrace.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant specific secrets",order=2,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster.
	// Set to true if you want to skip certification validation checks.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// Set custom proxy settings either directly or from a secret with the field proxy.
	// Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Proxy *DynaKubeProxy `json:"proxy,omitempty"`

	// Adds custom RootCAs from a configmap. Put the certificate under certs within your configmap.
	// Note: Applies only to Dynatrace Operator and OneAgent, not to ActiveGate.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trusted CAs",order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ConfigMap"}
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Sets a network zone for the OneAgent and ActiveGate pods.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`

	// Defines a custom pull secret in case you use a private registry when pulling images from the Dynatrace environment.
	// To define a custom pull secret and learn about the expected behavior, see Configure customPullSecret
	// (https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#custompullsecret).
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom PullSecret",order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will create the corresponding
	// VirtualService and ServiceEntry objects to allow access to the Dynatrace Cluster from the OneAgent or ActiveGate.
	// Disabled by default.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Istio automatic management",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	EnableIstio bool `json:"enableIstio,omitempty"`

	// Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject.
	// For more information, see Configure monitoring for namespaces and pods (https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// General configuration about OneAgent instances.
	// You can't enable more than one module (classicFullStack, cloudNativeFullStack, hostMonitoring, or applicationMonitoring).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	OneAgent OneAgentSpec `json:"oneAgent,omitempty"`

	// General configuration about ActiveGate instances.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ActiveGate",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ActiveGate ActiveGateSpec `json:"activeGate,omitempty"`

	// Configuration for Routing
	// +kubebuilder:deprecatedversion:warning="Deprecated property"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Routing"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Routing RoutingSpec `json:"routing,omitempty"`

	// Configuration for Kubernetes Monitoring
	// +kubebuilder:deprecatedversion:warning="Deprecated property"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Kubernetes Monitoring"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	KubernetesMonitoring KubernetesMonitoringSpec `json:"kubernetesMonitoring,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKubeList contains a list of DynaKube
// +kubebuilder:object:root=true
type DynaKubeList struct { // nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynaKube `json:"items"`
}

func init() {
	v1beta1.SchemeBuilder.Register(&DynaKube{}, &DynaKubeList{})
}
