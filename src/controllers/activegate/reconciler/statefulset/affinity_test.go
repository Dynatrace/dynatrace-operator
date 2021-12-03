package statefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAffinity(t *testing.T) {
	affinitySpec := affinity()
	nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	assert.Equal(t, 1, len(nodeSelectorTerms))
	assert.Contains(t, nodeSelectorTerms, kubernetesArchOsSelectorTerm())
}
