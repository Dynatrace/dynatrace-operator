package activegate

type ValueSource struct {
	// Custom properties value.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// Custom properties secret.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom properties secret",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}
