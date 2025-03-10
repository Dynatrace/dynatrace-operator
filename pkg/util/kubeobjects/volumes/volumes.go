package volumes

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func GetByName(vol []corev1.Volume, volumeName string) (*corev1.Volume, error) {
	for _, mount := range vol {
		if mount.Name == volumeName {
			return &mount, nil
		}
	}

	return nil, errors.Errorf(`Cannot find volume "%s" in the provided slice (len %d)`,
		volumeName, len(vol),
	)
}

func IsIn(vol []corev1.Volume, volumeName string) bool {
	for _, v := range vol {
		if v.Name == volumeName {
			return true
		}
	}

	return false
}
