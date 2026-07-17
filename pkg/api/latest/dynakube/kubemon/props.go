package kubemon

import corev1 "k8s.io/api/core/v1"

const (
	KubeMonAvailableConditionType = "KubernetesMonitoringAvailable"

	OperandNameSuffix = "-kubemon"

	ServiceAccountName = "dynatrace-activegate"
)

// KubeMon wraps Spec and Status for ergonomic access via dk.KubernetesMonitoring().
type KubeMon struct {
	*Spec
	*Status

	name string
}

func (km *Spec) IsEnabled() bool {
	return km != nil
}

// SetName seeds the DynaKube name onto the wrapper.
func (km *KubeMon) SetName(name string) {
	km.name = name
}

func (km *Spec) GetServiceAccountName() string {
	return ServiceAccountName
}

func (km *KubeMon) GetStatefulSetName() string {
	return km.name + OperandNameSuffix
}

func (km *KubeMon) GetConnectionInfoConfigMapName() string {
	return km.name + OperandNameSuffix + "-connection-info"
}

func (km *KubeMon) GetTenantSecretName() string {
	return km.name + OperandNameSuffix + "-tenant-secret"
}

func (km *KubeMon) GetAuthTokenSecretName() string {
	return km.name + OperandNameSuffix + "-authtoken-secret"
}

func (km *Spec) GetPullPolicy() corev1.PullPolicy {
	if km == nil {
		return ""
	}

	return corev1.PullPolicy(km.ImagePullPolicy)
}

// GetCustomImage returns the user-provided image override, or "" if unset.
func (km *Spec) GetCustomImage() string {
	if km == nil {
		return ""
	}

	return km.Image
}
