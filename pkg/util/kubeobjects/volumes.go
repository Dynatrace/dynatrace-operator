package kubeobjects

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func GetVolumeByName(volumes []corev1.Volume, volumeName string) (*corev1.Volume, error) {
	for _, volume := range volumes {
		if volume.Name == volumeName {
			return &volume, nil
		}
	}
	return nil, errors.Errorf(`Cannot find volume "%s" in the provided slice (len %d)`,
		volumeName, len(volumes),
	)
}

func GetVolumeMountByName(mounts []corev1.VolumeMount, volumeName string) (*corev1.VolumeMount, error) {
	for _, mount := range mounts {
		if mount.Name == volumeName {
			return &mount, nil
		}
	}
	return nil, errors.Errorf(`Cannot find volume mount "%s" in the provided slice (len %d)`,
		volumeName, len(mounts),
	)
}

func IsVolumeMountPresent(volumeMounts []corev1.VolumeMount, neededMount corev1.VolumeMount) bool {
	for _, volumeMount := range volumeMounts {
		if volumeMount == neededMount {
			return true
		}
	}
	return false
}
