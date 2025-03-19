package metadata

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	metacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/metadata"
	corev1 "k8s.io/api/core/v1"
)

func addInjectedEnv(container *corev1.Container) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentInjectedEnv, Value: "true"},
	)
}

func addWorkloadInfoEnvs(container *corev1.Container, workload *metacommon.WorkloadInfo) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentWorkloadKindEnv, Value: workload.Kind},
		corev1.EnvVar{Name: consts.EnrichmentWorkloadNameEnv, Value: workload.Name},
	)
}

func addDTClusterEnvs(container *corev1.Container, clusterName, entityID string) {
	container.Env = append(container.Env,
		corev1.EnvVar{Name: consts.EnrichmentClusterNameEnv, Value: clusterName},
		corev1.EnvVar{Name: consts.EnrichmentClusterEntityIDEnv, Value: entityID},
	)
}
