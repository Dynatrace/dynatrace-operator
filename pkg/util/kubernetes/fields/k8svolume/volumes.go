package k8svolume

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func Contains(vol []corev1.Volume, volumeName string) bool {
	for _, v := range vol {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}

func FindByName(volumes []corev1.Volume, volumeName string) (*corev1.Volume, error) {
	for _, volume := range volumes {
		if volume.Name == volumeName {
			return &volume, nil
		}
	}

	return nil, errors.Errorf(`Cannot find volume "%s" in the provided slice (len %d)`,
		volumeName, len(volumes),
	)
}
