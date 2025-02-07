package env

import (
	"encoding/json"
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	Tolerations  = "TOLERATIONS"
	NodeName     = "KUBE_NODE_NAME"
	CSIDataDir   = "CSI_DATA_DIR"
	PodNamespace = "POD_NAMESPACE"
	PodName      = "POD_NAME"
)

func FindEnvVar(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVars {
		if envVar.Name == name {
			// returning reference to env var to ease later manipulation of it
			return &envVars[i]
		}
	}

	return nil
}

func IsIn(envVars []corev1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}

	return false
}

func AddOrUpdate(envVars []corev1.EnvVar, desiredEnvVar corev1.EnvVar) []corev1.EnvVar {
	targetEnvVar := FindEnvVar(envVars, desiredEnvVar.Name)
	if targetEnvVar != nil {
		*targetEnvVar = desiredEnvVar
	} else {
		envVars = append(envVars, desiredEnvVar)
	}

	return envVars
}

func NewEnvVarSourceForField(fieldPath string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath}}
}

func DefaultNamespace() string {
	namespace := os.Getenv(PodNamespace)

	if namespace == "" {
		return "dynatrace"
	}

	return namespace
}

func GetNodeName() string {
	return os.Getenv(NodeName)
}

func GetCSIDataDir() string {
	return os.Getenv(CSIDataDir)
}

func GetTolerations() ([]corev1.Toleration, error) {
	var tolerations []corev1.Toleration

	raw := os.Getenv(Tolerations)
	if raw == "" {
		return tolerations, nil
	}

	err := json.Unmarshal([]byte(os.Getenv(Tolerations)), &tolerations)

	return tolerations, err
}
