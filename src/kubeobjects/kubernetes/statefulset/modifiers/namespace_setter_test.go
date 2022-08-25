package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNameSetter(t *testing.T) {
	t.Run("Set namespace", func(t *testing.T) {
		namespace := "aaa"

		b := statefulset.Builder{}
		b.AddModifier(
			NamespaceSetter{Namespace: namespace},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Override namespace", func(t *testing.T) {
		namespace0 := "aaa"
		namespace1 := "bbb"
		b := statefulset.Builder{}
		b.AddModifier(NamespaceSetter{Namespace: namespace0})
		b.AddModifier(NamespaceSetter{Namespace: namespace1})

		actual := b.Build()
		expected := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace1,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
