package statefulset

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAffinity(t *testing.T) {
	t.Run("default selector terms", func(t *testing.T) {
		affinitySpec := Affinity()
		nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

		assert.Equal(t, 1, len(nodeSelectorTerms))
		assert.Contains(t, nodeSelectorTerms, kubernetesArchOsSelectorTerm())
	})
	t.Run("affinity without OS term", func(t *testing.T) {
		affinitySpec := AffinityWithoutArch()

		assert.NotNil(t, affinitySpec)

		nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

		assert.Equal(t, 1, len(nodeSelectorTerms))
		assert.Contains(t, nodeSelectorTerms, kubernetesOsSelectorTerm())
	})
}
