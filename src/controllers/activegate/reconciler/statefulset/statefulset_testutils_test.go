package statefulset

import v1 "k8s.io/api/core/v1"

func mountPathIsIn(volumeMounts []v1.VolumeMount, mountPathToCheck string) bool {
	for _, volMount := range volumeMounts {
		if volMount.MountPath == mountPathToCheck {
			return true
		}
	}
	return false
}

func volumeIsDefined(volumes []v1.Volume, volumeMountNameToCheck string) bool {
	for _, vol := range volumes {
		if vol.Name == volumeMountNameToCheck {
			return true
		}
	}
	return false
}

func portIsIn(ports []v1.ContainerPort, portToCheck int32) bool {
	for _, port := range ports {
		if port.ContainerPort == portToCheck {
			return true
		}
	}
	return false
}

func envVarIsIn(envVars []v1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}
	return false
}
