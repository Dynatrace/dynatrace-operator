package consts

const (
	InjectionFailurePolicyEnv = "FAILURE_POLICY"

	ContainerInfoEnv = "CONTAINER_INFO"

	K8sNodeNameEnv    = "K8S_NODE_NAME"
	K8sPodNameEnv     = "K8S_PODNAME"
	K8sPodUIDEnv      = "K8S_PODUID"
	K8sBasePodNameEnv = "K8S_BASEPODNAME"
	K8sNamespaceEnv   = "K8S_NAMESPACE"
	K8sClusterIDEnv   = "K8S_CLUSTER_ID"

	SharedMountPath            = "/var/lib/dynatrace"
	SharedVolumeName           = "dt-share"
	ConfigVolumeName           = "injection-config"
	SharedDirMount             = "/mnt/share"
	SharedConfigConfigDirMount = "/mnt/config"
)
