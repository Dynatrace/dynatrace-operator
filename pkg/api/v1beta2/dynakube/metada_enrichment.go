package dynakube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MetaDataEnrichment struct {
	// The namespaces where you want Dynatrace Operator to inject enrichment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Enables MetaDataEnrichment.
	// Enabled by default.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="MetaDataEnrichment",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled *bool `json:"enabled" fieldalignment:"8"`
}
