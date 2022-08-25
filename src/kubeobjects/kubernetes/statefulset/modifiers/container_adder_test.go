package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestContainerAdder(t *testing.T) {
	t.Run("Add container", func(t *testing.T) {
		container := corev1.Container{Name: "sample container", Image: "busybox"}

		b := statefulset.Builder{}
		b.AddModifier(
			ContainerAdder{container: container},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{container},
					},
				},
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Add 2 containers", func(t *testing.T) {
		container0 := corev1.Container{Name: "sample-container-0", Image: "busybox"}
		container1 := corev1.Container{Name: "sample-container-1", Image: "busybox"}

		b := statefulset.Builder{}
		b.AddModifier(
			ContainerAdder{container: container0},
			ContainerAdder{container: container1},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{container0, container1},
					},
				},
			},
		}
		assert.Equal(t, expected, actual)
	})
}
