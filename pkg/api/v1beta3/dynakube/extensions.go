package dynakube

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type ExtensionsSpec struct {
	// +kubebuilder:validation:Optional
	Prometheus PrometheusSpec `json:"prometheus,omitempty"`
}

type TemplatesSpec struct {

	// +kubebuilder:validation:Optional
	OpenTelemetryCollector OpenTelemetryCollectorSpec `json:"openTelemetryCollector,omitempty"`
	// +kubebuilder:validation:Optional
	ExtensionExecutionController ExtensionExecutionControllerSpec `json:"extensionExecutionController,omitempty"`
}

type PrometheusSpec struct {
	Enabled bool `json:"enabled"`
}

type ExtensionExecutionControllerSpec struct {

	// Defines storage device
	// +kubebuilder:validation:Optional
	PersistentVolumeClaim *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`

	// Adds additional labels for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the ExtensionExecutionController pods
	Annotations map[string]string `json:"annotations,omitempty"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// +kubebuilder:validation:Optional
	TlsRefName string `json:"tlsRefName,omitempty"`

	// Define resources' requests and limits for single ExtensionExecutionController pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Determines retention policy
	// +kubebuilder:validation:Optional
	PersistentVolumeClaimRetentionPolicy *appsv1.PersistentVolumeClaimRetentionPolicyType `json:"persistentVolumeClaimRetentionPolicy,omitempty"`

	// Set tolerations for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds TopologySpreadConstraints for the ExtensionExecutionController pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

type OpenTelemetryCollectorSpec struct {

	// Adds additional labels for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the OtelCollector pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Number of replicas for your OtelCollector
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef ImageRefSpec `json:"imageRef,omitempty"`

	// +kubebuilder:validation:Optional
	TlsRefName string `json:"tlsRefName,omitempty"`

	// Define resources' requests and limits for single OtelCollector pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Set tolerations for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds TopologySpreadConstraints for the OtelCollector pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

type ImageRefSpec struct {
	// Custom image repository
	// +kubebuilder:example:="docker.io/dynatrace/image-name"
	Repository string `json:"repository,omitempty"`

	// Indicates a tag of the image to use
	Tag string `json:"tag,omitempty"`
}
