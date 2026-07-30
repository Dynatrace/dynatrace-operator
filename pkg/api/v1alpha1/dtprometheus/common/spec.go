// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=true

// Spec holds the pod-level settings shared by all DtPrometheus components.
type Spec struct {
	// Number of replicas for the component. At least 2 is recommended for
	// failure tolerance.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// Overrides the default component image (full reference including tag or digest).
	// When omitted the operator uses the image provided via the image endpoint.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// Image pull policy for the component pods.
	// +kubebuilder:validation:Optional
	PullPolicy image.PullPolicy `json:"pullPolicy,omitempty"`

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
