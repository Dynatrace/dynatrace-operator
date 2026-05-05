package daemonset

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

func getInitArgs(dk dynakube.DynaKube) []string {
	baseArgs := []string{
		fmt.Sprintf("-p k8s.cluster.uid=$(%s)", clusterUIDEnv),
		fmt.Sprintf("-p k8s.node.name=$(%s)", nodeNameEnv),
		fmt.Sprintf("-c k8s_fullpodname $(%s)", podNameEnv),
		fmt.Sprintf("-c k8s_poduid $(%s)", podUIDEnv),
		fmt.Sprintf("-c k8s_basepodname $(%s)", basePodNameEnv),
		fmt.Sprintf("-c k8s_namespace $(%s)", namespaceNameEnv),
		fmt.Sprintf("-c k8s_node_name $(%s)", nodeNameEnv),
		fmt.Sprintf("-c k8s_cluster_id $(%s)", clusterUIDEnv),
		"-c k8s_containername " + containerName,
		"-l " + dtLogVolumeMountPath,
	}

	if isMEConfigured(dk) {
		baseArgs = append(baseArgs, fmt.Sprintf("-p k8s.cluster.name=$(%s)", clusterNameEnv), fmt.Sprintf("-p dt.entity.kubernetes_cluster=$(%s)", entityEnv))
	}

	attrs := dk.GetResourceAttributes()
	for _, key := range slices.Sorted(maps.Keys(attrs)) {
		baseArgs = append(baseArgs, fmt.Sprintf("-p %s=%s", key, attrs[key]))
	}

	return append(baseArgs, dk.LogMonitoring().Template().Args...)
}
