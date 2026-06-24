package k8svolume

import (
	corev1 "k8s.io/api/core/v1"
)

func Contains(volumes []corev1.Volume, volumeName string) bool {
	for _, v := range volumes {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}

func FindByName(volumes []corev1.Volume, volumeName string) *corev1.Volume {
	for _, volume := range volumes {
		if volume.Name == volumeName {
			return &volume
		}
	}

	return nil
}
