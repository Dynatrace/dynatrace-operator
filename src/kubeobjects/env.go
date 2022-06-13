package kubeobjects

import corev1 "k8s.io/api/core/v1"

func FindEnvVar(envVarList []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVarList {
		if envVar.Name == name {
			// returning reference to env var to ease later manipulation of it
			return &envVarList[i]
		}
	}
	return nil
}

func EnvVarIsIn(envVars []corev1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}
	return false
}

func NewEnvVarSourceForField(fieldPath string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath}}
}
