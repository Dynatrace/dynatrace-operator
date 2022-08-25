package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestVolumeAdder(t *testing.T) {
	t.Run("Add volume", func(t *testing.T) {
		volume := corev1.Volume{Name: "sample"}

		b := statefulset.Builder{}
		b.AddModifier(
			VolumeAdder{volume: volume},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{volume},
					},
				},
			},
		}
		assert.Equal(t, expected, actual)
	})
	t.Run("Add 2 containers", func(t *testing.T) {
		volume0 := corev1.Volume{Name: "sample-volume0-0"}
		volume1 := corev1.Volume{Name: "sample-volume0-1"}

		b := statefulset.Builder{}
		b.AddModifier(
			VolumeAdder{volume: volume0},
			VolumeAdder{volume: volume1},
		)

		actual := b.Build()
		expected := appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{volume0, volume1},
					},
				},
			},
		}
		assert.Equal(t, expected, actual)
	})
}
