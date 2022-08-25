package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLabelsSetter(t *testing.T) {
	t.Run("Set labels", func(t *testing.T) {
		labels := Labels{"a": "b", "c": "d"}

		b := statefulset.Builder{}
		b.AddModifier(
			LabelsSetter{Labels: labels},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override labels", func(t *testing.T) {
		labels0 := Labels{"a": "b", "c": "d"}
		labels1 := Labels{"aa": "b", "cc": "d"}
		b := statefulset.Builder{}
		b.AddModifier(LabelsSetter{Labels: labels0})
		b.AddModifier(LabelsSetter{Labels: labels1})

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels1,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
