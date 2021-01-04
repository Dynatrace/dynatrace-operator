package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type OneAgentCodeModuleSpec struct {
	BaseOneAgentSpec `json:",inline"`

	// Enables Kubernetes Monitoring
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Kubernetes Monitoring"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	// Optional: Custom code modules OneAgent docker image
	// In case you have the docker image for the oneagent in a custom docker registry you need to provide it here
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=false
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonicx.ui:text"
	Image string `json:"image,omitempty"`

	// Optional: The version of the oneagent to be used
	// Default (if nothing set): latest
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent version"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonicx.ui:text"
	AgentVersion string `json:"agentVersion,omitempty"`

	// Optional: define resources requests and limits for the initContainer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Resource Requirements"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Optional: defines the C standard library used
	// Can be set to "musl" to use musl instead of glibc
	// If set to anything else but "musl", glibc is used
	// If a pod is annotated with the "oneagent.dynatrace.com/flavor" annotation, the value from the annotation will be used
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="C standard Library"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:select:default,urn:alm:descriptor:com.tectonic.ui:select:musl"
	Flavor string `json:"flavor,omitempty"`
}

type OneAgentCodeModuleStatus struct {
	BaseOneAgentStatus `json:",inline"`
}