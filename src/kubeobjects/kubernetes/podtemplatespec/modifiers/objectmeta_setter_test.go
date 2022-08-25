package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/podtemplatespec"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObjectMetaSetter(t *testing.T) {
	t.Run("Set objectmeta", func(t *testing.T) {

		om := metav1.ObjectMeta{Name: "asd"}

		b := podtemplatespec.Builder{}
		b.AddModifier(
			ObjectMetaSetter{ObjectMeta: om},
		)

		actual := b.Build()
		expected := corev1.PodTemplateSpec{
			ObjectMeta: om,
		}
		assert.Equal(t, expected, actual)
	})
}
