// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSpec_GetReplicas(t *testing.T) {
	tests := []struct {
		name     string
		replicas *int32
		expected int32
	}{
		{"unset returns default", nil, DefaultReplicas},
		{"set returns value", new(int32(5)), 5},
		{"explicit zero returns zero", new(int32(0)), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &Spec{Replicas: tt.replicas}
			assert.Equal(t, tt.expected, spec.GetReplicas())
		})
	}
}

func TestSpec_GetPullPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   image.PullPolicy
		expected corev1.PullPolicy
	}{
		{"unset", "", corev1.PullPolicy("")},
		{"always", image.PullPolicy("Always"), corev1.PullAlways},
		{"if not present", image.PullPolicy("IfNotPresent"), corev1.PullIfNotPresent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &Spec{PullPolicy: tt.policy}
			assert.Equal(t, tt.expected, spec.GetPullPolicy())
		})
	}
}

func TestSpec_PassthroughGetters(t *testing.T) {
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("6Gi")},
	}
	affinity := &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{}}
	tolerations := []corev1.Toleration{{Key: "monitoring", Operator: corev1.TolerationOpEqual, Value: "true"}}
	spreadConstraints := []corev1.TopologySpreadConstraint{{MaxSkew: 1, TopologyKey: "kubernetes.io/hostname"}}
	nodeSelector := map[string]string{"kubernetes.io/os": "linux"}
	annotations := map[string]string{"annotation": "a"}
	labels := map[string]string{"label": "l"}

	spec := &Spec{
		Image:                     "docker.io/dynatrace/dynatrace-otel-collector:latest",
		Resources:                 resources,
		NodeSelector:              nodeSelector,
		Affinity:                  affinity,
		Tolerations:               tolerations,
		TopologySpreadConstraints: spreadConstraints,
		PriorityClassName:         "high-priority",
		Annotations:               annotations,
		Labels:                    labels,
	}

	assert.Equal(t, "docker.io/dynatrace/dynatrace-otel-collector:latest", spec.GetImage())
	assert.Equal(t, resources, spec.GetResources())
	assert.Equal(t, nodeSelector, spec.GetNodeSelector())
	assert.Same(t, affinity, spec.GetAffinity())
	assert.Equal(t, tolerations, spec.GetTolerations())
	assert.Equal(t, spreadConstraints, spec.GetTopologySpreadConstraints())
	assert.Equal(t, "high-priority", spec.GetPriorityClassName())
	assert.Equal(t, annotations, spec.GetAnnotations())
	assert.Equal(t, labels, spec.GetLabels())
}

func TestSpec_ZeroValueGetters(t *testing.T) {
	spec := &Spec{}

	assert.Empty(t, spec.GetImage())
	assert.Equal(t, corev1.ResourceRequirements{}, spec.GetResources())
	assert.Nil(t, spec.GetNodeSelector())
	assert.Nil(t, spec.GetAffinity())
	assert.Nil(t, spec.GetTolerations())
	assert.Nil(t, spec.GetTopologySpreadConstraints())
	assert.Empty(t, spec.GetPriorityClassName())
	assert.Nil(t, spec.GetAnnotations())
	assert.Nil(t, spec.GetLabels())
}
