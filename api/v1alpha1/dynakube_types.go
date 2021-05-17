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

	// Optional: Pull secret for your private registry
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom PullSecret",order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Disable certificate validation checks for installer download and API communication
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Certificate Check",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// Optional: Set custom proxy settings either directly or from a secret with the field 'proxy'
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

	// General configuration about ActiveGate instances
	ActiveGate ActiveGateSpec `json:"activeGate,omitempty"`

	// General configuration about OneAgent instances
	OneAgent OneAgentSpec `json:"oneAgent,omitempty"`

	// Configuration for CodeModules Monitoring
	CodeModules CodeModulesSpec `json:"codeModules,omitempty"`

	// Configuration for Infra Monitoring
	InfraMonitoring FullStackSpec `json:"infraMonitoring,omitempty"`

	// Configuration for ClassicFullStack Monitoring
	ClassicFullStack FullStackSpec `json:"classicFullStack,omitempty"`

	//  Configuration for Routing
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Routing"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	RoutingSpec RoutingSpec `json:"routing,omitempty"`

	//  Configuration for Kubernetes Monitoring
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Kubernetes Monitoring"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	KubernetesMonitoringSpec KubernetesMonitoringSpec `json:"kubernetesMonitoring,omitempty"`
}

type ActiveGateSpec struct {
	// Optional: the ActiveGate container image. Defaults to the latest ActiveGate image provided by the Docker Registry
	// implementation from the Dynatrace environment set as API URL.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Disable automatic restarts of OneAgent pods in case a new version is available
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Automatically update Agent"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	AutoUpdate *bool `json:"autoUpdate,omitempty"`
}

type OneAgentSpec struct {
	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent version",order=11,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Version string `json:"version,omitempty"`

	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=12,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Disable automatic restarts of OneAgent pods in case a new version is available
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Automatically update Agent",order=13,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	AutoUpdate *bool `json:"autoUpdate,omitempty"`
}

type CodeModulesSpec struct {
	// Enables code modules monitoring
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CodeModules Monitoring",order=14,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	// Optional: define resources requests and limits for the initContainer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=15,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: use OneAgent binaries from volume
	Volume corev1.VolumeSource `json:"volume,omitempty"`
}

type FullStackSpec struct {
	// Enables FullStack Monitoring
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="FullStack Monitoring",order=16,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	// Node selector to control the selection of nodes (optional)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=17,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: Defines the time to wait until OneAgent pod is ready after update - default 300 sec
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Wait seconds until ready",order=19,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:number"}
	WaitReadySeconds *uint16 `json:"waitReadySeconds,omitempty"`

	// Optional: define resources requests and limits for single pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=20,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Arguments to the OneAgent installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent installer arguments",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OneAgent environment variable installer arguments",order=22,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Optional: If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class name",order=23,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:PriorityClass"}
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Optional: Sets DNS Policy for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DNS Policy",order=24,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Optional: set custom Service Account Name used with OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account name",order=25,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ServiceAccount"}
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Optional: Adds additional labels for the OneAgent pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=26,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: Runs the OneAgent Pods as unprivileged (Early Adopter)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Use unprivileged mode",order=27,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	UseUnprivilegedMode *bool `json:"useUnprivilegedMode,omitempty"`

	// Defines if you want to use the immutable image or the installer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Use immutable image",order=28,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`
}

type RoutingSpec struct {
	CapabilityProperties `json:",inline"`
}

type KubernetesMonitoringSpec struct {
	CapabilityProperties `json:",inline"`
}

// CapabilityProperties is a struct which can be embedded by ActiveGate capabilities
// Such as KubernetesMonitoring or Routing
// It encapsulates common properties
type CapabilityProperties struct {
	// Enables Capability
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capability",order=29,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	// Amount of replicas for your DynaKube
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: Set activation group for ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Activation group",order=31,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Group string `json:"group,omitempty"`

	// Optional: Add a custom properties file by providing it as a value or reference it from a secret
	// If referenced from a secret, make sure the key is called 'customProperties'
	CustomProperties *DynaKubeValueSource `json:"customProperties,omitempty"`

	// Optional: define resources requests and limits for single ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=34,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: Node selector to control the selection of nodes
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=35,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the ActiveGatePods pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=36,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Optional: Adds additional labels for the ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=37,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// Optional: Adds additional arguments for the ActiveGate instances
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Arguments",order=38,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the ActiveGate
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=39,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +listType=set
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Environment variables"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Optional: set custom Service Account Name used with ActiveGate pods
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account name",order=40,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:ServiceAccount"}
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type DynaKubeValueSource struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}

type DynaKubeProxy struct {
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"valueFrom,omitempty"`
}

// DynaKubeStatus defines the observed state of DynaKube
// +k8s:openapi-gen=true
type DynaKubeStatus struct {
	// Defines the current state (Running, Updating, Error, ...)
	Phase DynaKubePhaseType `json:"phase,omitempty"`

	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`

	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`

	// Credentials used to connect back to Dynatrace.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API and PaaS Tokens"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	Tokens string `json:"tokens,omitempty"`

	// LastClusterVersionProbeTimestamp indicates when the cluster's version was last checked
	LastClusterVersionProbeTimestamp *metav1.Time `json:"lastClusterVersionProbeTimestamp,omitempty"`

	// EnvironmentID contains the environment ID corresponding to the API URL
	EnvironmentID string `json:"environmentID,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ActiveGate ActiveGateStatus `json:"activeGate,omitempty"`

	OneAgent OneAgentStatus `json:"oneAgent,omitempty"`
}

type VersionStatus struct {
	// ImageHash contains the last image hash seen.
	ImageHash string `json:"imageHash,omitempty"`

	// Version contains the version to be deployed.
	Version string `json:"version,omitempty"`

	// LastUpdateProbeTimestamp defines the last timestamp when the querying for updates have been done
	LastUpdateProbeTimestamp *metav1.Time `json:"lastUpdateProbeTimestamp,omitempty"`
}

type ActiveGateStatus struct {
	VersionStatus `json:",inline"`
}

type OneAgentStatus struct {
	VersionStatus `json:",inline"`

	// UseImmutableImage is set when an immutable image is currently in use
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`

	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	// LastHostsRequestTimestamp indicates the last timestamp the Operator queried for hosts
	LastHostsRequestTimestamp *metav1.Time `json:"lastHostsRequestTimestamp,omitempty"`
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

type DynaKubePhaseType string

const (
	Running   DynaKubePhaseType = "Running"
	Deploying DynaKubePhaseType = "Deploying"
	Error     DynaKubePhaseType = "Error"
)

// SetPhase sets the status phase on the DynaKube object
func (dk *DynaKubeStatus) SetPhase(phase DynaKubePhaseType) bool {
	upd := phase != dk.Phase
	dk.Phase = phase
	return upd
}

// SetPhaseOnError fills the phase with the Error value in case of any error
func (dk *DynaKubeStatus) SetPhaseOnError(err error) bool {
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

// DynaKube is the Schema for the DynaKube API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dynakubes,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.status.tokens`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:csv:customresourcedefinitions:displayName="Dynatrace DynaKube"
// +operator-sdk:csv:customresourcedefinitions:resources={{StatefulSet,v1,},{DaemonSet,v1,},{Pod,v1,}}
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
