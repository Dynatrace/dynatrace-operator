// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	corev1 "k8s.io/api/core/v1"
)

type OpenTelemetryCollectorSpec struct {

	// Adds additional labels for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Adds additional annotations to the OtelCollector pods
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Number of replicas for your OtelCollector
	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas"`

	// Overrides the default image
	// +kubebuilder:validation:Optional
	ImageRef image.Ref `json:"imageRef,omitzero"`

	// +kubebuilder:validation:Optional
	TLSRefName string `json:"tlsRefName,omitempty"`

	// Define resources' requests and limits for single OtelCollector pod
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitzero"`

	// Node selector to control the selection of nodes for the OtelCollector pods
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Set tolerations for the OtelCollector pods
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Adds TopologySpreadConstraints for the OtelCollector pods
	// +kubebuilder:validation:Optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}
