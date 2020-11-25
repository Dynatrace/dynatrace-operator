package v1alpha1

type OneAgentAPMSpec struct {
	Enabled bool `json:"enabled,omitempty"`

	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/activegate:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/activegate for OpenShift
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Image string `json:"image,omitempty"`

	UseImmutabelImage bool   `json:"useImmutableImage,omitempty"`
	AgentVersion      string `json:"agentVersion,omitempty"`
	EnableIstio       bool   `json:"enableIstio,omitempty"`
}
