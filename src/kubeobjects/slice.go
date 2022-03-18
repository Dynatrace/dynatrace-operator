package kubeobjects

import corev1 "k8s.io/api/core/v1"

const (
	ReadOnlyMountPath  = true
	ReadWriteMountPath = false
)

func MountPathIsIn(volumeMounts []corev1.VolumeMount, mountPathToCheck string) bool {
	for _, volMount := range volumeMounts {
		if volMount.MountPath == mountPathToCheck {
			return true
		}
	}
	return false
}

func MountPathIsReadOnlyOrReadWrite(volumeMounts []corev1.VolumeMount, mountPathToCheck string, mode bool) bool {
	for _, volMount := range volumeMounts {
		if volMount.MountPath == mountPathToCheck && volMount.ReadOnly == mode {
			return true
		}
	}
	return false
}

func VolumeIsDefined(volumes []corev1.Volume, volumeNameToCheck string) bool {
	for _, vol := range volumes {
		if vol.Name == volumeNameToCheck {
			return true
		}
	}
	return false
}

func VolumeMountIsDefined(volumeMounts []corev1.VolumeMount, volumeMountNameToCheck string) bool {
	for _, vol := range volumeMounts {
		if vol.Name == volumeMountNameToCheck {
			return true
		}
	}
	return false
}

func PortIsIn(ports []corev1.ContainerPort, portToCheck int32) bool {
	for _, port := range ports {
		if port.ContainerPort == portToCheck {
			return true
		}
	}
	return false
}

func EnvVarIsIn(envVars []corev1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}
	return false
}
