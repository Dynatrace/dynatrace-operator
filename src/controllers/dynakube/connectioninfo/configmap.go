package connectioninfo

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateTestConnectionInfoConfigMap(tenantUUID string, dynakube *dynatracev1beta1.DynaKube) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: dynakube.OneAgentConnectionInfoConfigMapName(), Namespace: dynakube.Namespace},
		Data: map[string]string{
			TenantUUIDName: tenantUUID,
		},
	}
}
