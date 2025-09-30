package metadataenrichment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnrichmentRuleType string

const (
	LabelRule          EnrichmentRuleType = "LABEL"
	AnnotationRule     EnrichmentRuleType = "ANNOTATION"
	Annotation         string             = "metadata.dynatrace.com"
	Prefix                                = Annotation + "/"
	namespaceKeyPrefix string             = "k8s.namespace."
)

type MetadataEnrichment struct {
	*Spec
	*Status
}

// +kubebuilder:object:generate=true
type Spec struct {
	// Enables MetadataEnrichment, `false` by default.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="MetaDataEnrichment",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:booleanSwitch"
	Enabled *bool `json:"enabled,omitempty"`

	// The namespaces where you want Dynatrace Operator to inject enrichment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

type EnrichmentRule struct {
	Type   EnrichmentRuleType `json:"type,omitempty"`
	Source string             `json:"source,omitempty"`
	Target string             `json:"target,omitempty"`
}
