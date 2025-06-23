package volumes

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/mounts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	ConfigVolumeName    = "dynatrace-config"
	InitConfigMountPath = "/mnt/config"
	InitConfigSubPath   = "config"
	ConfigMountPath     = "/var/lib/dynatrace"

	InputVolumeName    = "dynatrace-input"
	InitInputMountPath = "/mnt/input"
)

func AddConfigVolume(pod *corev1.Pod) {
	if volumes.IsIn(pod.Spec.Volumes, ConfigVolumeName) {
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: ConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func AddConfigVolumeMount(container *corev1.Container) {
	if mounts.IsPathIn(container.VolumeMounts, ConfigMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: ConfigMountPath,
			SubPath:   filepath.Join(InitConfigSubPath, container.Name),
		},
	)
}

func AddInitConfigVolumeMount(container *corev1.Container) {
	if mounts.IsPathIn(container.VolumeMounts, InitConfigMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      ConfigVolumeName,
			MountPath: InitConfigMountPath,
			SubPath:   InitConfigSubPath,
		},
	)
}

func AddInputVolume(pod *corev1.Pod) {
	if volumes.IsIn(pod.Spec.Volumes, InputVolumeName) {
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: InputVolumeName,
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
		},
	)
}

func AddInitInputVolumeMount(container *corev1.Container) {
	if mounts.IsPathIn(container.VolumeMounts, InitInputMountPath) {
		return
	}

	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      InputVolumeName,
			MountPath: InitInputMountPath,
			ReadOnly:  true,
		},
	)
}
