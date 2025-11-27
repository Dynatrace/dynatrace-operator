package k8smount

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func Find(mounts []corev1.VolumeMount, volumeName string) (*corev1.VolumeMount, error) {
	for _, mount := range mounts {
		if mount.Name == volumeName {
			return &mount, nil
		}
	}

	return nil, errors.Errorf(`Cannot find volume mount "%s" in the provided slice (len %d)`,
		volumeName, len(mounts),
	)
}

func ContainsPath(mounts []corev1.VolumeMount, path string) bool {
	for _, vm := range mounts {
		if vm.MountPath == path {
			return true
		}
	}

	return false
}

func Contains(mounts []corev1.VolumeMount, volumeName string) bool {
	for _, m := range mounts {
		if m.Name == volumeName {
			return true
		}
	}

	return false
}

func Append(mounts []corev1.VolumeMount, vm ...corev1.VolumeMount) []corev1.VolumeMount {
	for _, v := range vm {
		if !ContainsPath(mounts, v.MountPath) {
			mounts = append(mounts, v)
		}
	}

	return mounts
}
