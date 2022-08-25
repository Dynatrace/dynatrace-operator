package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodTemplateSpecSetter(t *testing.T) {
	t.Run("Set PodTemplateSpec", func(t *testing.T) {
		pts := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: "asd",
			},
		}

		b := statefulset.Builder{}
		b.AddModifier(
			PodTemplateSpecSetter{PodTemplateSpec: pts},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: pts,
			},
		}
		assert.Equal(t, expected, actual)
	})
}
