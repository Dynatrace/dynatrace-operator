// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
	corev1 "k8s.io/api/core/v1"
)

// PodSpec holds the parts of the corev1.PodTemplateSpec that DTPrometheus
// components surface for users to configure. The full corev1 spec is not
// exposed, to limit conflicts with what the operator needs to configure for
// proper functioning.
type PodSpec struct {
	// Number of replicas for the component. At least 2 is recommended for
	// failure tolerance.
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Overrides the default component image (full reference including tag or digest).
	// When omitted the operator uses the latest available image from the public Dynatrace registry.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// Image pull policy for the component pods.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=IfNotPresent;Always;Never
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Command-line arguments for the main application container.
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`

	// Resource requests and limits for each component pod.
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitzero"`

	// Node selector to control the selection of nodes for the component pods.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Full affinity rules (nodeAffinity, podAffinity, podAntiAffinity) for the
	// component pods.
	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations for the component pods.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Topology spread constraints for the component pods.
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

	// PriorityClass assigned to the component pods so they are not evicted under
	// node pressure.
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Extra annotations merged into the pod template metadata.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Extra labels merged into the pod template metadata.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

func (s PodSpec) SanitizedArgs() []string {
	return sanitize.CommandLineArgs(s.Args)
}
