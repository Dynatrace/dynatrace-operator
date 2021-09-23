package statefulset

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestAffinity(t *testing.T) {
// 	affinitySpec := affinity(&statefulSetProperties{
// 		majorKubernetesVersion: "1",
// 		minorKubernetesVersion: "20",
// 	})
// 	affinitySpecWithBeta := affinity(&statefulSetProperties{
// 		majorKubernetesVersion: "1",
// 		minorKubernetesVersion: "13",
// 	})
// 	nodeSelectorTerms := affinitySpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
// 	betaNodeSelectorTerms := affinitySpecWithBeta.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

// 	assert.Equal(t, 1, len(nodeSelectorTerms))
// 	assert.Equal(t, 2, len(betaNodeSelectorTerms))

// 	assert.Contains(t, nodeSelectorTerms, kubernetesArchOsSelectorTerm())
// 	assert.NotContains(t, nodeSelectorTerms, kubernetesBetaArchOsSelectorTerm())

// 	assert.Contains(t, betaNodeSelectorTerms, kubernetesArchOsSelectorTerm())
// 	assert.Contains(t, betaNodeSelectorTerms, kubernetesBetaArchOsSelectorTerm())
// }
