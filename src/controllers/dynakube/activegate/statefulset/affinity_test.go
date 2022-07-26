package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAffinity(t *testing.T) {
	t.Run("default selector terms", func(t *testing.T) {
		affinitySpec := affinity()
		nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

		assert.Equal(t, 1, len(nodeSelectorTerms))
		assert.Contains(t, nodeSelectorTerms, kubernetesArchOsSelectorTerm())
	})
	t.Run("affinity without OS term", func(t *testing.T) {
		affinitySpec := affinityWithoutArch()

		assert.NotNil(t, affinitySpec)

		nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

		assert.Equal(t, 1, len(nodeSelectorTerms))
		assert.Contains(t, nodeSelectorTerms, kubeobjects.AffinityNodeRequirement())
	})
}
