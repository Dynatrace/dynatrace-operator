package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenericModifier(t *testing.T) {
	t.Run("Run modifier of statefulset", func(t *testing.T) {
		const name = "modified"
		b := statefulset.Builder{}
		b.AddModifier(
			GenericModifier{func(sts *appsv1.StatefulSet) {
				sts.Name = name
			}},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
