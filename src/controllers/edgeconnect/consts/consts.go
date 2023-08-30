package consts

const (
	EdgeConnectContainerName               = "edge-connect"
	EdgeConnectServiceAccountName          = "dynatrace-edgeconnect"
	EdgeConnectMountPath                   = "/etc/edge_connect"
	EdgeConnectVolumeMountName             = "oauth-secret"
	AnnotationEdgeConnectContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + EdgeConnectContainerName
)
