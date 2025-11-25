package k8senv

import (
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	NodeName     = "KUBE_NODE_NAME"
	CSIDataDir   = "CSI_DATA_DIR"
	PodNamespace = "POD_NAMESPACE"
	PodName      = "POD_NAME"
)

func Find(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVars {
		if envVar.Name == name {
			// returning reference to env var to ease later manipulation of it
			return &envVars[i]
		}
	}

	return nil
}

func FindCaseInsensitive(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for i, envVar := range envVars {
		if strings.EqualFold(envVar.Name, name) {
			// returning reference to env var to ease later manipulation of it
			return &envVars[i]
		}
	}

	return nil
}

func Contains(envVars []corev1.EnvVar, envVarToCheck string) bool {
	for _, envVar := range envVars {
		if envVar.Name == envVarToCheck {
			return true
		}
	}

	return false
}

func Append(envVars []corev1.EnvVar, envVarToAppend corev1.EnvVar) ([]corev1.EnvVar, bool) {
	added := false

	if !Contains(envVars, envVarToAppend.Name) {
		envVars = append(envVars, envVarToAppend)
		added = true
	}

	return envVars, added
}

func AddOrUpdate(envVars []corev1.EnvVar, desiredEnvVar corev1.EnvVar) []corev1.EnvVar {
	targetEnvVar := Find(envVars, desiredEnvVar.Name)
	if targetEnvVar != nil {
		*targetEnvVar = desiredEnvVar
	} else {
		envVars = append(envVars, desiredEnvVar)
	}

	return envVars
}

func NewSourceForField(fieldPath string) *corev1.EnvVarSource {
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
