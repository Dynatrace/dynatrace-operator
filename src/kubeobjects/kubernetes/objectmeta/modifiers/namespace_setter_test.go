package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceSetter(t *testing.T) {
	t.Run("Set namespace", func(t *testing.T) {
		namespace := "aaa"

		b := objectmeta.Builder{}
		b.AddModifier(
			NamespaceSetter{Namespace: namespace},
		)

		actual := b.Build()
		expected := metav1.ObjectMeta{
			Namespace: namespace,
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override namespace", func(t *testing.T) {
		namespace0 := "aaa"
		namespace1 := "bbb"
		b := objectmeta.Builder{}
		b.AddModifier(NamespaceSetter{Namespace: namespace0})
		b.AddModifier(NamespaceSetter{Namespace: namespace1})

		actual := b.Build()
		expected := metav1.ObjectMeta{
			Namespace: namespace1,
		}
		assert.Equal(t, expected, actual)
	})
}
