package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameSetter(t *testing.T) {
	t.Run("Set name", func(t *testing.T) {
		name := "aaa"

		b := statefulset.Builder{}
		b.AddModifier(
			NameSetter{Name: name},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override name", func(t *testing.T) {
		name0 := "aaa"
		name1 := "bbb"
		b := statefulset.Builder{}
		b.AddModifier(NameSetter{Name: name0})
		b.AddModifier(NameSetter{Name: name1})

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name1,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
