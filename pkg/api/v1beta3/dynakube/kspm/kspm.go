package kspm

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	TokenSecretKey = "kspm-token"
)

type Kspm struct {
	*Spec
	*NodeConfigurationCollectorSpec

	name string
}

// +kubebuilder:object:generate=true

type Spec struct {
	Enabled bool `json:"enabled"`
}

// +kubebuilder:object:generate=true

type NodeConfigurationCollectorSpec struct {

	// Define the NodeConfigurationCollector daemonSet updateStrategy
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`
	// Adds additional labels for the NodeConfigurationCollector pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations for the NodeConfigurationCollector pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Specify the node selector that controls on which nodes NodeConfigurationCollector pods will be deployed.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef image.Ref `json:"imageRef,omitempty"`

	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Define resources' requests and limits for single NodeConfigurationCollector pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Define the nodeAffinity for the DaemonSet of the NodeConfigurationCollector
	// +kubebuilder:validation:Optional
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// Set tolerations for the NodeConfigurationCollector pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Set additional arguments to the NodeConfigurationCollector pods
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`

	// Set additional environment variables for the NodeConfigurationCollector pods
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}
