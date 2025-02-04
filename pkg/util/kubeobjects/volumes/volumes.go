package volumes

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

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

func IsIn(mounts []corev1.VolumeMount, volumeName string) bool {
	for _, vm := range mounts {
		if vm.Name == volumeName {
			return true
		}
	}

	return false
}
