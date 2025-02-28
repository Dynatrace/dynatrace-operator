package dynakube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MetadataEnrichment struct {
	// Enables MetadataEnrichment, `false` by default.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="MetaDataEnrichment",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled *bool `json:"enabled,omitempty"`

	// The namespaces where you want Dynatrace Operator to inject enrichment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}
