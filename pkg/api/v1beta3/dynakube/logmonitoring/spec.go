package logmonitoring

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	corev1 "k8s.io/api/core/v1"
)

type LogMonitoring struct {
	*Spec
	*TemplateSpec

	name string
}

// +kubebuilder:object:generate=true
type Spec struct {
}

// +kubebuilder:object:generate=true
type TemplateSpec struct {
	// Add custom annotations to the LogMonitoring pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Add custom labels to the LogMonitoring pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Node selector to control the selection of nodes for the LogMonitoring pods
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Overrides the default image for the LogMonitoring pods
	// +kubebuilder:validation:Optional
	ImageRef image.Ref `json:"imageRef,omitempty"`

	// Sets DNS Policy for the LogMonitoring pods
	// +kubebuilder:validation:Optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Assign a priority class to the LogMonitoring pods. By default, no class is set
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// The SecComp Profile that will be configured in order to run in secure computing mode for the LogMonitoring pods
	// +kubebuilder:validation:Optional
	SecCompProfile string `json:"secCompProfile,omitempty"`

	// Define resources' requests and limits for all the LogMonitoring pods
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Set tolerations for the LogMonitoring pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Set additional arguments to the LogMonitoring main container
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`
}
