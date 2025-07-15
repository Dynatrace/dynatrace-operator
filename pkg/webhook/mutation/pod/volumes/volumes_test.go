package volumes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestAddInputVolume(t *testing.T) {
	t.Run("two projected volumes added to pod spec as single volume source", func(t *testing.T) {
		pod := &corev1.Pod{}

		AddInputVolume(pod)

		assert.Len(t, pod.Spec.Volumes, 1)

		assert.Equal(t, corev1.Volume{
			Name: "dynatrace-input",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: consts.BootstrapperInitSecretName,
								},
								Optional: ptr.To(false),
							},
						},
						{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: consts.BootstrapperInitCertsSecretName,
								},
								Optional: ptr.To(true),
							},
						},
					},
				},
			},
		}, pod.Spec.Volumes[0])
	})
}
