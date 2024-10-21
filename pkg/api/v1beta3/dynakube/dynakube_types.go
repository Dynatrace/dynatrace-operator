// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1beta3
package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: Move these conditions related consts to a place where they are used, so we don't bloat this package further.
const (
	// TokenConditionType identifies the token validity condition.
	TokenConditionType string = "Tokens"

	// APITokenConditionType identifies the API Token validity condition.
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition.
	PaaSTokenConditionType string = "PaaSToken"

	// DataIngestTokenConditionType identifies the DataIngest Token validity condition.
	DataIngestTokenConditionType string = "DataIngestToken"
)

// Possible reasons for ApiToken and PaaSToken conditions.
const (
	// ReasonTokenReady is set when a token has passed verifications.
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenError is set when an unknown error has been found when verifying the token.
	ReasonTokenError string = "TokenError"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dynakubes,scope=Namespaced,categories=dynatrace,shortName={dk,dks}
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:csv:customresourcedefinitions:displayName="Dynatrace DynaKube"
// +operator-sdk:csv:customresourcedefinitions:resources={{StatefulSet,v1,},{DaemonSet,v1,},{Pod,v1,}}
type DynaKube struct {
	metav1.TypeMeta `json:",inline"`

	Status            DynaKubeStatus `json:"status,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DynaKubeSpec `json:"spec,omitempty"`
}

// DynaKubeSpec defines the desired state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeSpec struct { //nolint:revive
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Set custom proxy settings either directly or from a secret with the field proxy.
	// Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Proxy *value.Source `json:"proxy,omitempty"`

	// General configuration about OneAgent instances.
	// You can't enable more than one module (classicFullStack, cloudNativeFullStack, hostMonitoring, or applicationMonitoring).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	OneAgent OneAgentSpec `json:"oneAgent,omitempty"`

	// Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address.
	// For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="APIURL is immutable, please delete the CR and then apply new one"
	// +kubebuilder:validation:MaxLength=128
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API URL",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Name of the secret holding the tokens used for connecting to Dynatrace.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant specific secrets",order=2,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Adds custom RootCAs from a configmap. Put the certificate under certs within your configmap.
	// Note: Applies to Dynatrace Operator, OneAgent and ActiveGate.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trusted CAs",order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ConfigMap"}
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Sets a network zone for the OneAgent and ActiveGate pods.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`

	// Defines a custom pull secret in case you use a private registry when pulling images from the Dynatrace environment.
	// To define a custom pull secret and learn about the expected behavior, see Configure customPullSecret
	// (https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#custompullsecret).
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom PullSecret",order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// +kubebuilder:validation:Optional
	Templates TemplatesSpec `json:"templates,omitempty"`

	// General configuration about the LogMonitoring feature.
	// +kubebuilder:validation:Optional
	LogMonitoring *logmonitoring.Spec `json:"logMonitoring,omitempty"`

	// Configuration for Metadata Enrichment.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Metadata Enrichment",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	MetadataEnrichment MetadataEnrichment `json:"metadataEnrichment,omitempty"`

	// General configuration about ActiveGate instances.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ActiveGate",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ActiveGate activegate.Spec `json:"activeGate,omitempty"`

	// Configuration for thresholding Dynatrace API requests.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=15
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Dynatrace API Request Threshold",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	DynatraceApiRequestThreshold int `json:"dynatraceApiRequestThreshold,omitempty"`

	// Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster.
	// Set to true if you want to skip certification validation checks.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will create the corresponding
	// VirtualService and ServiceEntry objects to allow access to the Dynatrace Cluster from the OneAgent or ActiveGate.
	// Disabled by default.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Istio automatic management",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	EnableIstio bool `json:"enableIstio,omitempty"`

	// +kubebuilder:validation:Optional
	Extensions ExtensionsSpec `json:"extensions,omitempty"`

	// General configuration about the KSPM feature.
	// +kubebuilder:validation:Optional
	Kspm kspm.Spec `json:"kspm,omitempty"`
}

type TemplatesSpec struct {
	// +kubebuilder:validation:Optional
	KspmNodeConfigurationCollector kspm.NodeConfigurationCollectorSpec `json:"kspmNodeConfigurationCollector,omitempty"`
	// +kubebuilder:validation:Optional
	ExtensionExecutionController ExtensionExecutionControllerSpec `json:"extensionExecutionController,omitempty"`
	// Low-level configuration options for the LogMonitoring feature.
	// +kubebuilder:validation:Optional
	LogMonitoring logmonitoring.TemplateSpec `json:"logMonitoring,omitempty"`
	// +kubebuilder:validation:Optional
	OpenTelemetryCollector OpenTelemetryCollectorSpec `json:"openTelemetryCollector,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKubeList contains a list of DynaKube
// +kubebuilder:object:root=true
type DynaKubeList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynaKube `json:"items"`
}

func init() {
	v1beta3.SchemeBuilder.Register(&DynaKube{}, &DynaKubeList{})
}
