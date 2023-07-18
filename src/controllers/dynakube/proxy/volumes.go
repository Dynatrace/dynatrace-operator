package proxy

import (
	"github.com/Dynatrace/dynatrace-operator/src/config"
	corev1 "k8s.io/api/core/v1"
)

func PrepareVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      config.ProxySecretVolumeName,
		MountPath: config.ProxySecretMountPath,
	}
}
