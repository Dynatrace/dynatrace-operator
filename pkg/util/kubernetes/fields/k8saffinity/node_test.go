package k8saffinity

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAffinity(t *testing.T) {
	affinity := NewMultiArchNodeAffinity()

	require.NotNil(t, affinity)
	require.NotNil(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	require.Len(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, 1)

	matchExpression := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
	assert.Equal(t, matchExpression, nodeAffinityRequirementsForArches(arch.AMDImage, arch.ARMImage, arch.PPCLEImage, arch.S390Image))
	assert.Contains(t, matchExpression, linuxRequirement())
}

func TestAffinityForArches(t *testing.T) {
	expectedArches := []string{"arch1", "arch2", "arch3"}
	affinity := nodeAffinityForArches(expectedArches...)

	require.NotNil(t, affinity)
	require.NotNil(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	require.Len(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, 1)

	matchExpression := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions
	assert.Equal(t, matchExpression, nodeAffinityRequirementsForArches(expectedArches...))
	assert.Contains(t, matchExpression, linuxRequirement())
}

func linuxRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      kubernetesOS,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{arch.DefaultImageOS},
	}
}
