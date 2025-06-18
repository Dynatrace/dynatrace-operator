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

	DynatraceAPICapability = ActiveGateCapability{
		DisplayName:  "dynatrace-api",
		ShortName:    "dynatrace-api",
		ArgumentName: "restInterface",
	}
)

var ActiveGateDisplayNames = map[CapabilityDisplayName]struct{}{
	RoutingCapability.DisplayName:       {},
	KubeMonCapability.DisplayName:       {},
	MetricsIngestCapability.DisplayName: {},
	DynatraceAPICapability.DisplayName:  {},
}

type ActiveGateSpec struct {

	// Adds additional annotations to the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations",order=27,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Annotations map[string]string `json:"annotations,omitempty"`

	// The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used.
	// `server.p12`: certificate+key pair in pkcs12 format
	// `password`: passphrase to read server.p12
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TlsSecretName",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	TLSSecretName string `json:"tlsSecretName,omitempty"`

	// Sets DNS Policy for the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DNS Policy",order=24,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Assign a priority class to the ActiveGate pods. By default, no class is set.
	// For details, see Pod Priority and Preemption. (https://dt-url.net/n8437bl)
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class name",order=23,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:PriorityClass"}
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Defines the ActiveGate pod capabilities
	// Possible values:
	//	- `routing` enables OneAgent routing.
	//	- `kubernetes-monitoring` enables Kubernetes API monitoring.
	//	- `metrics-ingest` opens the metrics ingest endpoint on the DynaKube ActiveGate and redirects all pods to it.
	//	- `dynatrace-api` enables calling the Dynatrace API via ActiveGate.
	Capabilities []CapabilityDisplayName `json:"capabilities,omitempty"`

	CapabilityProperties `json:",inline"`
}

// CapabilityProperties is a struct which can be embedded by ActiveGate capabilities
// Such as KubernetesMonitoring or Routing
// It encapsulates common properties.
type CapabilityProperties struct {

	// Add a custom properties file by providing it as a value or reference it from a secret
	// +kubebuilder:validation:Optional
	// If referenced from a secret, make sure the key is called `customProperties`
	CustomProperties *DynaKubeValueSource `json:"customProperties,omitempty"`

	// Specify the node selector that controls on which nodes ActiveGate will be deployed.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=35,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Your defined labels for ActiveGate pods in order to structure workloads as desired.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",order=37,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Labels map[string]string `json:"labels,omitempty"`

	// Use a custom ActiveGate image. Defaults to the latest ActiveGate image provided by the registry on the tenant
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Set activation group for ActiveGate
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Activation group",order=31,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Group string `json:"group,omitempty"`

	// Resource settings for ActiveGate container.
	// Consumption of the ActiveGate heavily depends on the workload to monitor. Adjust values accordingly.
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

	// Adds TopologySpreadConstraints to the ActiveGate pods
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="topologySpreadConstraints",order=40,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	// Amount of replicas for your ActiveGates
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas int32 `json:"replicas,omitempty"`
}
