package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestPodManagementPolicySetter(t *testing.T) {
	t.Run("Set replicas", func(t *testing.T) {
		const pmp = appsv1.ParallelPodManagement

		b := statefulset.Builder{}
		b.AddModifier(
			PodManagementPolicySetter{PodManagementPolicy: pmp},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				PodManagementPolicy: pmp,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
