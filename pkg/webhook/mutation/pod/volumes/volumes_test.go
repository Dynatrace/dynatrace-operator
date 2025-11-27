package volumes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestAddConfigVolume(t *testing.T) {
	t.Run("should add config volume to pod without annotation", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "alpine",
					},
				},
				InitContainers: []corev1.Container{
					{
						Name:  "init-container",
						Image: "alpine",
					},
				},
			},
		}

		AddConfigVolume(pod)

		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, corev1.Volume{
			Name:         "dynatrace-config",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		}, pod.Spec.Volumes[0])
	})

	t.Run("should add config volume to pod with annotation for emptyDir size limit value", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"volume.dynatrace.com/dynatrace-config": "300Mi",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "alpine",
					},
				},
				InitContainers: []corev1.Container{
					{
						Name:  "init-container",
						Image: "alpine",
					},
				},
			},
		}

		AddConfigVolume(pod)

		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, corev1.Volume{
			Name: "dynatrace-config",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{
				SizeLimit: ptr.To(resource.MustParse("300Mi")),
			}},
		}, pod.Spec.Volumes[0])
	})
}

func TestAddConfigVolumeMount(t *testing.T) {
	t.Run("should add common config volume mount if split mounts is disabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
			Pod:      &corev1.Pod{},
		}

		AddConfigVolumeMount(container, request)

		assert.Len(t, container.VolumeMounts, 1)
		assert.True(t, HasCommonConfigVolumeMounts(container))
	})

	t.Run("should add split mounts for oneagent if split mounts is enabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.InjectionSplitMounts: "true",
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
			Pod:      pod,
		}

		AddConfigVolumeMount(container, request)

		assert.Len(t, container.VolumeMounts, 1)

		assert.True(t, HasSplitOneAgentMounts(container))
		assert.False(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should add split mounts for both if split mounts is enabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.InjectionSplitMounts: "true",
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
			Pod:      pod,
		}

		AddConfigVolumeMount(container, request)

		assert.Len(t, container.VolumeMounts, 4)

		assert.True(t, HasSplitOneAgentMounts(container))
		assert.True(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should add split mounts for metadataenrichment if enabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.InjectionSplitMounts: "true",
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
			Pod:      pod,
		}

		AddConfigVolumeMount(container, request)

		assert.Len(t, container.VolumeMounts, 3)
		assert.False(t, HasSplitOneAgentMounts(container))
		assert.True(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should add split mounts for metadataenrichment if classicfullstack is enabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.InjectionSplitMounts: "true",
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
			Pod:      pod,
		}

		AddConfigVolumeMount(container, request)

		assert.Len(t, container.VolumeMounts, 3)
		assert.False(t, HasSplitOneAgentMounts(container))
		assert.True(t, HasSplitEnrichmentMounts(container))
	})
}
