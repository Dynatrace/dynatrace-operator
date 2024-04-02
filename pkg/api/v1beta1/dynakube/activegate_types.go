package dynakube

import (
	corev1 "k8s.io/api/core/v1"
)

type CapabilityDisplayName string

type ActiveGateCapability struct {

	// The name of the capability known by the user, mainly used in the CR
	DisplayName CapabilityDisplayName

	// The name used for marking the pod for given capability
	ShortName string

	// The string passed to the active gate image to enable a given capability
	ArgumentName string
}

var (
	RoutingCapability = ActiveGateCapability{
		DisplayName:  "routing",
		ShortName:    "routing",
		ArgumentName: "MSGrouter",
	}

	KubeMonCapability = ActiveGateCapability{
		DisplayName:  "kubernetes-monitoring",
		ShortName:    "kubemon",
		ArgumentName: "kubernetes_monitoring",
	}

	MetricsIngestCapability = ActiveGateCapability{
		DisplayName:  "metrics-ingest",
		ShortName:    "metrics-ingest",
		ArgumentName: "metrics_ingest",
	}

	DynatraceApiCapability = ActiveGateCapability{
		DisplayName:  "dynatrace-api",
		ShortName:    "dynatrace-api",
		ArgumentName: "restInterface",
	}
)

var ActiveGateDisplayNames = map[CapabilityDisplayName]struct{}{
	RoutingCapability.DisplayName:       {},
	KubeMonCapability.DisplayName:       {},
	MetricsIngestCapability.DisplayName: {},
	DynatraceApiCapability.DisplayName:  {},
}

type ActiveGateSpec struct {

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
	CustomProperties *DynaKubeValueSource `json:"customProperties,omitempty"`

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
