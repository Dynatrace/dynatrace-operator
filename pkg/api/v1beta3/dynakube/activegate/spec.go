package activegate

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	corev1 "k8s.io/api/core/v1"
)

type CapabilityDisplayName string

type Capability struct {

	// The name of the capability known by the user, mainly used in the CR
	DisplayName CapabilityDisplayName

	// The name used for marking the pod for given capability
	ShortName string

	// The string passed to the active gate image to enable a given capability
	ArgumentName string
}

var (
	RoutingCapability = Capability{
		DisplayName:  "routing",
		ShortName:    "routing",
		ArgumentName: "MSGrouter",
	}

	KubeMonCapability = Capability{
		DisplayName:  "kubernetes-monitoring",
		ShortName:    "kubemon",
		ArgumentName: "kubernetes_monitoring",
	}

	MetricsIngestCapability = Capability{
		DisplayName:  "metrics-ingest",
		ShortName:    "metrics-ingest",
		ArgumentName: "metrics_ingest",
	}

	DynatraceApiCapability = Capability{
		DisplayName:  "dynatrace-api",
		ShortName:    "dynatrace-api",
		ArgumentName: "restInterface",
	}
)

var CapabilityDisplayNames = map[CapabilityDisplayName]struct{}{
	RoutingCapability.DisplayName:       {},
	KubeMonCapability.DisplayName:       {},
	MetricsIngestCapability.DisplayName: {},
	DynatraceApiCapability.DisplayName:  {},
}

type ActiveGate struct {
	*Spec
	*Status
}

// dependencies is a collection of possible other feature/components that need an ActiveGate, but is not directly configured in the ActiveGate section.
type dependencies struct {
	extensions bool
}

func (d dependencies) Any() bool {
	return d.extensions
}

// +kubebuilder:object:generate=true

type Spec struct {

	// Adds additional annotations to the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations",order=27,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Annotations map[string]string `json:"annotations,omitempty"`

	name   string
	apiUrl string

	// The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used.
	// server.p12: certificate+key pair in pkcs12 format
	// password: passphrase to read server.p12
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TlsSecretName",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	TlsSecretName string `json:"tlsSecretName,omitempty"`

	// Sets DNS Policy for the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DNS Policy",order=24,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the StatefulSet.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class name",order=23,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:PriorityClass"}
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)
	Capabilities []CapabilityDisplayName `json:"capabilities,omitempty"`

	CapabilityProperties `json:",inline"`

	enabledDependencies dependencies
}

// +kubebuilder:object:generate=true

// CapabilityProperties is a struct which can be embedded by ActiveGate capabilities
// Such as KubernetesMonitoring or Routing
// It encapsulates common properties.
type CapabilityProperties struct {

	// Add a custom properties file by providing it as a value or reference it from a secret
	// +kubebuilder:validation:Optional
	// If referenced from a secret, make sure the key is called 'customProperties'
	CustomProperties *value.Source `json:"customProperties,omitempty"`

	// Node selector to control the selection of nodes
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=35,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Adds additional labels for the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=37,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// The ActiveGate container image. Defaults to the latest ActiveGate image provided by the registry on the tenant
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Set activation group for ActiveGate
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Activation group",order=31,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Group string `json:"group,omitempty"`

	// Define resources requests and limits for single ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements",order=34,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Set tolerations for the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tolerations",order=36,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// List of environment variables to set for the ActiveGate
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=39,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Environment variables"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Adds TopologySpreadConstraints for the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="topologySpreadConstraints",order=40,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	// Amount of replicas for your ActiveGates
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas int32 `json:"replicas,omitempty"`
}
