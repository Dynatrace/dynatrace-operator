package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func getInitArgs(dk dynakube.DynaKube) []string {
	baseArgs := []string{
		fmt.Sprintf("-p k8s.cluster.name=$(%s)", clusterNameEnv),
		fmt.Sprintf("-p k8s.cluster.uid=$(%s)", clusterUIDEnv),
		fmt.Sprintf("-p k8s.node.name=$(%s)", nodeNameEnv),
		fmt.Sprintf("-p dt.entity.kubernetes_cluster=$(%s)", entityEnv),
		fmt.Sprintf("-c k8s_fullpodname $(%s)", podNameEnv),
		fmt.Sprintf("-c k8s_poduid $(%s)", podUIDEnv),
		fmt.Sprintf("-c k8s_containername %s", containerName),                       //nolint:perfsprint
		fmt.Sprintf("-c k8s_basepodname %s", dk.LogMonitoring().GetDaemonSetName()), //nolint:perfsprint
		fmt.Sprintf("-c k8s_namespace $(%s)", namespaceNameEnv),
		fmt.Sprintf("-c k8s_node_name $(%s)", nodeNameEnv),
		fmt.Sprintf("-c k8s_cluster_id $(%s)", clusterUIDEnv),
	}

	return append(baseArgs, dk.LogMonitoring().Args...)
}
