// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package common

import corev1 "k8s.io/api/core/v1"

// DefaultReplicas is the replica count used when a component does not set one.
const DefaultReplicas int32 = 2

// GetReplicas returns the configured replica count, or the default when unset.
func (spec *Spec) GetReplicas() int32 {
	if spec.Replicas == nil {
		return DefaultReplicas
	}

	return *spec.Replicas
}

// GetImage returns the custom image override, or an empty string when the
// operator should fall back to the image provided via the image endpoint.
func (spec *Spec) GetImage() string {
	return spec.Image
}

// GetPullPolicy returns the image pull policy for the component pods.
func (spec *Spec) GetPullPolicy() corev1.PullPolicy {
	return corev1.PullPolicy(spec.PullPolicy)
}

// GetResources returns the resource requests and limits for the component pods.
func (spec *Spec) GetResources() corev1.ResourceRequirements {
	return spec.Resources
}

// GetNodeSelector returns the node selector for the component pods.
func (spec *Spec) GetNodeSelector() map[string]string {
	return spec.NodeSelector
}

// GetAffinity returns the affinity rules for the component pods.
func (spec *Spec) GetAffinity() *corev1.Affinity {
	return spec.Affinity
}

// GetTolerations returns the tolerations for the component pods.
func (spec *Spec) GetTolerations() []corev1.Toleration {
	return spec.Tolerations
}

// GetTopologySpreadConstraints returns the topology spread constraints for the component pods.
func (spec *Spec) GetTopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	return spec.TopologySpreadConstraints
}

// GetPriorityClassName returns the PriorityClass assigned to the component pods.
func (spec *Spec) GetPriorityClassName() string {
	return spec.PriorityClassName
}

// GetAnnotations returns the extra pod annotations for the component.
func (spec *Spec) GetAnnotations() map[string]string {
	return spec.Annotations
}

// GetLabels returns the extra pod labels for the component.
func (spec *Spec) GetLabels() map[string]string {
	return spec.Labels
}
