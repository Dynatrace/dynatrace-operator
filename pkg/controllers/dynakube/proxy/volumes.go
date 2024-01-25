package proxy

import (
	corev1 "k8s.io/api/core/v1"
)

func BuildVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      SecretVolumeName,
		MountPath: SecretMountPath,
	}
}
