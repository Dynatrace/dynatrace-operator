package value

// +kubebuilder:object:generate=true
type Source struct {
	// Raw value for given property.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="property value",order=32,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Value string `json:"value,omitempty"`

	// Name of the secret to get the property from.
	// +nullable
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="property secret name",order=33,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:io.kubernetes:Secret"}
	ValueFrom string `json:"valueFrom,omitempty"`
}
