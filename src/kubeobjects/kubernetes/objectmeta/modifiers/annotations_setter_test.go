package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta"
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnnotationsSetter(t *testing.T) {
	t.Run("Set annotations", func(t *testing.T) {
		annotations := internalTypes.Annotations{"a": "b", "c": "d"}

		b := objectmeta.Builder{}
		b.AddModifier(
			AnnotationsSetter{Annotations: annotations},
		)

		actual := b.Build()
		expected := metav1.ObjectMeta{
			Annotations: annotations,
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override annotations", func(t *testing.T) {
		annotations0 := internalTypes.Annotations{"a": "b", "c": "d"}
		annotations1 := internalTypes.Annotations{"aa": "b", "cc": "d"}
		b := objectmeta.Builder{}
		b.AddModifier(AnnotationsSetter{Annotations: annotations0})
		b.AddModifier(AnnotationsSetter{Annotations: annotations1})

		actual := b.Build()
		expected := metav1.ObjectMeta{
			Annotations: annotations1,
		}
		assert.Equal(t, expected, actual)
	})
}
