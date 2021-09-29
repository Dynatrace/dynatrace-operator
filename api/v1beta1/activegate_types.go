package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

type ActiveGateCapability string

const (
	Routing    ActiveGateCapability = "routing"
	KubeMon    ActiveGateCapability = "kubernetes-monitoring"
	DataIngest ActiveGateCapability = "data-ingest"
)

type ActiveGateSpec struct {

	// Activegate capabilities enabled (routing, kubernetes-monitoring, data-ingest)
	Capabilities []ActiveGateCapability `json:"capabilities,omitempty"`

	// Amount of replicas for your DynaKube
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas *int32 `json:"replicas,omitempty"`

	CapabilityProperties `json:",inline"`

	// Optional: the name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used.
	// server.p12: certificate+key pair in pkcs12 format
	// password: passphrase to read server.p12
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TlsSecretName",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	TlsSecretName string `json:"tlsSecretName,omitempty"`
}

// CapabilityProperties is a struct which can be embedded by ActiveGate capabilities
// Such as KubernetesMonitoring or Routing
// It encapsulates common properties
type CapabilityProperties struct {
	// Amount of replicas for your DynaKube
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Replicas *int32 `json:"replicas,omitempty"`

	// Optional: the ActiveGate container image. Defaults to the latest ActiveGate image provided by the Docker Registry
	// implementation from the Dynatrace environment set as API URL.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

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
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Environment variables"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`
}
