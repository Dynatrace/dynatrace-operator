package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OperatorName = "dynatrace-operator"
)

// DynaKubeSpec defines the desired state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeSpec struct {
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Location of the Dynatrace API to connect to, including your specific environment ID
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API URL",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Credentials for the DynaKube to connect back to Dynatrace.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API and PaaS Tokens",order=2,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Disable certificate validation checks for API communication
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// Optional: Set custom proxy settings either directly or from a secret with the field 'proxy'
	Proxy *DynaKubeProxy `json:"proxy,omitempty"`

	// Optional: Adds custom RootCAs from a configmap
	// This property only affects certificates used to communicate with the Dynatrace API.
	// The property is not applied to the ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trusted CAs",order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ConfigMap"}
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Optional: Sets Network Zone for ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Network Zone",order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	NetworkZone string `json:"networkZone,omitempty"`

	// Optional: Pull secret for your private registry
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom PullSecret",order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// General configuration about ActiveGate instances
	ActiveGate ActiveGateSpec `json:"activeGate,omitempty"`

	// Enables Kubernetes Monitoring
	KubernetesMonitoringSpec KubernetesMonitoringSpec `json:"kubernetesMonitoring,omitempty"`
}

type ActiveGateSpec struct {
	// Optional: the ActiveGate container image. Defaults to the latest ActiveGate image provided by the Docker Registry
	// implementation from the Dynatrace environment set as API URL.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`
}

type KubernetesMonitoringSpec struct {
	// Enables Kubernetes Monitoring
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Kubernetes Monitoring",order=10,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	// Amount of replicas for your DynaKube
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=11,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: Set activation group for ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Activation group",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Group string `json:"group,omitempty"`

	// Optional: Add a custom properties file by providing it as a value or reference it from a secret
	// If referenced from a secret, make sure the key is called 'customProperties'
	CustomProperties *DynaKubeValueSource `json:"customProperties,omitempty"`

	// Optional: define resources requests and limits for single ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=15,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Node selector to control the selection of nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=16,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the ActiveGatePods pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=17,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: Adds additional labels for the ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: Adds additional arguments for the ActiveGate instances
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Arguments",order=19,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=20,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Optional: set custom Service Account Name used with ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account name",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ServiceAccount"}
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type DynaKubeValueSource struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties value",order=13,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties secret",order=14,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

type DynaKubeProxy struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy value",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Proxy from secret",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct {
	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Last Updated",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`

	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`

	// ActiveGateImageHash contains the last image hash seen.
	ActiveGateImageHash string `json:"activeGateImageHash,omitempty"`

	// ActiveGateImageVersion contains the version from the last image seen.
	ActiveGateImageVersion string `json:"activeGateImageVersion,omitempty"`

	// Credentials used to connect back to Dynatrace.
	Tokens string `json:"tokens,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dynakubes,scope=Namespaced
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.status.tokens`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:csv:customresourcedefinitions:displayName="Dynatrace DynaKube"
// +operator-sdk:csv:customresourcedefinitions:resources={{StatefulSet,v1,apps}}
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
