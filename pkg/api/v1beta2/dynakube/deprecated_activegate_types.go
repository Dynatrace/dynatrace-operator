package dynakube

// Deprecated: Use ActiveGateSpec instead.
type RoutingSpec struct {
	CapabilityProperties `json:",inline"`
	// Enables Capability
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capability",order=29,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`
}

// Deprecated: Use ActiveGateSpec instead.
type KubernetesMonitoringSpec struct {
	CapabilityProperties `json:",inline"`
	// Enables Capability
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Capability",order=29,xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled bool `json:"enabled,omitempty"`
}
