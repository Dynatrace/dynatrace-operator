package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnnotationsSetter(t *testing.T) {
	t.Run("Set annotations", func(t *testing.T) {
		annotations := Annotations{"a": "b", "c": "d"}

		b := statefulset.Builder{}
		b.AddModifier(
			AnnotationsSetter{Annotations: annotations},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override annotations", func(t *testing.T) {
		annotations0 := Annotations{"a": "b", "c": "d"}
		annotations1 := Annotations{"aa": "b", "cc": "d"}
		b := statefulset.Builder{}
		b.AddModifier(AnnotationsSetter{Annotations: annotations0})
		b.AddModifier(AnnotationsSetter{Annotations: annotations1})

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations1,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
