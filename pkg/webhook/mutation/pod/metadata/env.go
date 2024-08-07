package metadata

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

func addInjectedEnv(container *corev1.Container) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentInjectedEnv, Value: "true"},
	)
}

func addWorkloadInfoEnvs(container *corev1.Container, workload *workloadInfo) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentWorkloadKindEnv, Value: workload.kind},
		corev1.EnvVar{Name: consts.EnrichmentWorkloadNameEnv, Value: workload.name},
	)
}

func addClusterNameEnv(container *corev1.Container, clusterName string) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentClusterNameEnv, Value: clusterName},
	)
}
