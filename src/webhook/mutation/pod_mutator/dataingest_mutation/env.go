package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/config"
	corev1 "k8s.io/api/core/v1"
)

func addWorkloadInfoEnvs(container *corev1.Container, workload *workloadInfo) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: config.EnrichmentWorkloadKindEnv, Value: workload.kind},
		corev1.EnvVar{Name: config.EnrichmentWorkloadNameEnv, Value: workload.name},
		corev1.EnvVar{Name: config.EnrichmentInjectedEnv, Value: "true"},
	)
}
