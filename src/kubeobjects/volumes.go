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

func GetVolumeMountByName(volumeMounts []corev1.VolumeMount, volumeName string) (*corev1.VolumeMount, error) {
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == volumeName {
			return &volumeMount, nil
		}
	}
	return nil, errors.Errorf(`Cannot find volume mount "%s" in the provided slice (len %d)`,
		volumeName, len(volumeMounts),
	)
}
