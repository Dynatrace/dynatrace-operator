package metadataenrichment

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuleType string

const (
	LabelRule          RuleType = "LABEL"
	AnnotationRule     RuleType = "ANNOTATION"
	Annotation         string   = "metadata.dynatrace.com"
	Prefix                      = Annotation + "/"
	namespaceKeyPrefix string   = "k8s.namespace."
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

	// Define resources' requests and limits for the initContainer used for standalone metadata-enrichment.
	// Only respected when no OneAgent is injected.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Requirements"
	InitResources *corev1.ResourceRequirements `json:"initResources,omitempty"`
}

type Rule struct {
	Type   RuleType `json:"type,omitempty"`
	Source string   `json:"source,omitempty"`
	Target string   `json:"target,omitempty"`
}
