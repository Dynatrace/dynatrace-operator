package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/podtemplatespec"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPodSpecSetter(t *testing.T) {
	t.Run("Set objectmeta", func(t *testing.T) {
		ps := corev1.PodSpec{
			Hostname: "bla",
		}

		b := podtemplatespec.Builder{}
		b.AddModifier(
			PodSpecSetter{PodSpec: ps},
		)

		actual := b.Build()
		expected := corev1.PodTemplateSpec{
			Spec: ps,
		}
		assert.Equal(t, expected, actual)
	})
}
