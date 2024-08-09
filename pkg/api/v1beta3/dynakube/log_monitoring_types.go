package dynakube

import corev1 "k8s.io/api/core/v1"

type LogMonitoringSpec struct {
	Enabled bool `json:"enabled"`
}

type LogAgentSpec struct {
	// Add custom LogAgent annotations
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Add custom LogAgent labels
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Node selector to control the selection of nodes for the LogAgent pods
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// Sets DNS Policy for the ActiveGate pods
	// +kubebuilder:validation:Optional
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`

	// Assign a priority class to the LogAgent pods. By default, no class is set
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// The SecComp Profile that will be configured in order to run in secure computing mode
	// +kubebuilder:validation:Optional
	SecCompProfile string `json:"secCompProfile,omitempty"`

	// Define resources' requests and limits for single LogAgent pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Set tolerations for the LogAgent pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Set additional arguments to the LogAgent pods
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`
}
