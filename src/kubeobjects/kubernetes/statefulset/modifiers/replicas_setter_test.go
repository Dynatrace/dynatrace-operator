package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestReplicasSetter(t *testing.T) {
	t.Run("Set replicas", func(t *testing.T) {
		const replicasValue int32 = 42
		replicasPtr := address.Of(replicasValue)

		b := statefulset.Builder{}
		b.AddModifier(
			ReplicasSetter{Replicas: replicasPtr},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Replicas: replicasPtr,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
