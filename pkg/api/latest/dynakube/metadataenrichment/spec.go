package metadataenrichment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuleType string

const (
	LabelRule      RuleType = "LABEL"
	AnnotationRule RuleType = "ANNOTATION"

	K8sNamespaceLabelRule      RuleType = "K8S_NAMESPACE_LABEL"
	K8sNamespaceAnnotationRule RuleType = "K8S_NAMESPACE_ANNOTATION"
	// TODO: implement support for this type.
	CustomRule RuleType = "CUSTOM"

	Annotation         = "metadata.dynatrace.com"
	Prefix             = Annotation + "/"
	namespaceKeyPrefix = "k8s.namespace."
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

type Rule struct {
	Type   RuleType `json:"type,omitempty"`
	Source string   `json:"source,omitempty"`
	Target string   `json:"target,omitempty"`
}

// IsSupportedType returns true if a rule's type should be used for further processing.
func IsSupportedType(ruleType RuleType) bool {
	switch ruleType {
	case LabelRule,
		AnnotationRule,
		K8sNamespaceLabelRule,
		K8sNamespaceAnnotationRule:
		return true
	}

	return false
}
