package attributes

const (
	K8sContainerNameAttr = "k8s.container.name"
	K8sNodeNameEnv       = "K8S_NODE_NAME"
	K8sPodNameEnv        = "K8S_PODNAME"
	K8sPodUIDEnv         = "K8S_PODUID"

	K8sPodNameAttr       = "k8s.pod.name"
	K8sPodUIDAttr        = "k8s.pod.uid"
	K8sNodeNameAttr      = "k8s.node.name"
	K8sNamespaceNameAttr = "k8s.namespace.name"

	K8sClusterUIDAttr      = "k8s.cluster.uid"
	K8sClusterNameAttr     = "k8s.cluster.name"
	K8sDTClusterEntityAttr = "dt.entity.kubernetes_cluster"

	K8sWorkloadKindAttr = "k8s.workload.kind"
	K8sWorkloadNameAttr = "k8s.workload.name"
)
