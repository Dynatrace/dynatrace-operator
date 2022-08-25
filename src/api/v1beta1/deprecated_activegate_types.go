package v1beta1

// Deprecated: Use ActiveGateSpec instead
type RoutingSpec struct {
	// Enables Capability
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capability",order=29,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	CapabilityProperties `json:",inline"`
}

// Deprecated: Use ActiveGateSpec instead
type KubernetesMonitoringSpec struct {
	// Enables Capability
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capability",order=29,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`

	CapabilityProperties `json:",inline"`
}
