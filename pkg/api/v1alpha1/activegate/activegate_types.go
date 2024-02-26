// +kubebuilder:object:generate=true
// +groupName=dynatrace.com
// +versionName=v1alpha1
// +kubebuilder:validation:Optional
package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SpecBasis struct {
	Common   CommonSpec   `json:",inline"`
	Specific SpecificSpec `json:",inline"`
}

type CommonSpec struct {
	// Set custom proxy settings either directly or from a secret with the field proxy.
	// Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Proxy *dynatracev1beta1.DynaKubeProxy `json:"proxy,omitempty"`

	// Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address.
	// For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id).
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API URL",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Name of the secret holding the tokens used for connecting to Dynatrace.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant specific secrets",order=2,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Sets a network zone for the OneAgent and ActiveGate pods.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`

	// Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster.
	// Set to true if you want to skip certification validation checks.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`
}

type SpecificSpec struct {

	// Adds additional annotations to the ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations",order=27,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Annotations map[string]string `json:"annotations,omitempty"`

	// The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used.
	// server.p12: certificate+key pair in pkcs12 format
	// password: passphrase to read server.p12
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TlsSecretName",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	TlsSecretName string `json:"tlsSecretName,omitempty"`

	// Sets DNS Policy for the ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DNS Policy",order=24,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the StatefulSet.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class name",order=23,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:PriorityClass"}
	PriorityClassName string `json:"priorityClassName,omitempty"`

	CapabilityProperties `json:",inline"`

	// Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capabilities",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Capabilities []CapabilityDisplayName `json:"capabilities,omitempty"`
}

// CapabilityProperties is a struct which can be embedded by ActiveGate capabilities
// Such as KubernetesMonitoring or Routing
// It encapsulates common properties.
type CapabilityProperties struct {
	// Amount of replicas for your ActiveGates
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas *int32 `json:"replicas,omitempty"`

	// The ActiveGate container image. Defaults to the latest ActiveGate image provided by the registry on the tenant
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Set activation group for ActiveGate
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Activation group",order=31,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Group string `json:"group,omitempty"`

	// Add a custom properties file by providing it as a value or reference it from a secret
	// +optional
	// If referenced from a secret, make sure the key is called 'customProperties'
	CustomProperties *ValueSource `json:"customProperties,omitempty"`

	// Define resources requests and limits for single ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=34,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Node selector to control the selection of nodes
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=35,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Set tolerations for the ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=36,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds additional labels for the ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=37,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// List of environment variables to set for the ActiveGate
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=39,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Environment variables"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Adds TopologySpreadConstraints for the ActiveGate pods
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="topologySpreadConstraints",order=40,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveGate is the Schema for the ActiveGate API
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=activegates,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiServer",type=string,JSONPath=`.spec.apiServer`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:storageversion
type ActiveGate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ActiveGateStatus `json:"status,omitempty"`

	Spec SpecBasis `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveGateList contains a list of ActiveGate
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
type ActiveGateList struct { //nolint:revive
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActiveGate `json:"items"`
}

func init() {
	v1alpha1.SchemeBuilder.Register(&ActiveGate{}, &ActiveGateList{})
}
