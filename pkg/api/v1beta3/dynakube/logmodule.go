package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/common"
	corev1 "k8s.io/api/core/v1"
)

type LogModuleSpec struct {
	Enabled bool `json:"enabled"`
}

type LogModuleTemplateSpec struct {
	// Add custom annotations to the LogModule pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Add custom labels to the LogModule pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Node selector to control the selection of nodes for the LogModule pods
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Overrides the default image for the LogModule pods
	// +kubebuilder:validation:Optional
	ImageRef common.ImageRefSpec `json:"imageRef,omitempty"`

	// Sets DNS Policy for the LogModule pods
	// +kubebuilder:validation:Optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Assign a priority class to the LogModule pods. By default, no class is set
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// The SecComp Profile that will be configured in order to run in secure computing mode for the LogModule pods
	// +kubebuilder:validation:Optional
	SecCompProfile string `json:"secCompProfile,omitempty"`

	// Define resources' requests and limits for all the LogModule pods
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Set tolerations for the LogModule pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Set additional arguments to the LogModule main container
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`
}
