package volumes

import (
	corev1 "k8s.io/api/core/v1"
)

func IsIn(vol []corev1.Volume, volumeName string) bool {
	for _, v := range vol {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}
