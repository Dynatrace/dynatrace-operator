package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	corev1 "k8s.io/api/core/v1"
)

func addWorkloadInfoEnvs(container *corev1.Container, workload *workloadInfo) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: standalone.WorkloadKindEnv, Value: workload.kind},
		corev1.EnvVar{Name: standalone.WorkloadNameEnv, Value: workload.name},
		corev1.EnvVar{Name: standalone.DataIngestInjectedEnv, Value: "true"},
	)
}
