package consts

const (
	EdgeConnectUserProvisioned       = "user-provisioned"
	EdgeConnectContainerName         = "edge-connect"
	EdgeConnectServiceAccountName    = "dynatrace-edgeconnect"
	EdgeConnectMountPath             = "/etc/ssl"
	EdgeConnectCustomCertificateName = "certificate.cer"
	EdgeConnectCustomCAVolumeName    = "ca-certs"
	EdgeConnectConfigFileName        = "edgeConnect.yaml"
	EdgeConnectConfigPath            = "/" + EdgeConnectConfigFileName
	EdgeConnectConfigVolumeMountName = "edge-connect-config-yaml"
	EdgeConnectCAConfigMapKey        = "certs"

	EnvEdgeConnectName            = "EDGE_CONNECT_NAME"
	EnvEdgeConnectApiEndpointHost = "EDGE_CONNECT_API_ENDPOINT_HOST"
	EnvEdgeConnectOauthEndpoint   = "EDGE_CONNECT_OAUTH__ENDPOINT"
	EnvEdgeConnectOauthResource   = "EDGE_CONNECT_OAUTH__RESOURCE"
	EnvEdgeConnectRestrictHostsTo = "EDGE_CONNECT_RESTRICT_HOSTS_TO"

	KeyEdgeConnectOauthClientID     = "oauth-client-id"
	KeyEdgeConnectOauthClientSecret = "oauth-client-secret"
	KeyEdgeConnectOauthResource     = "oauth-client-resource"
	KeyEdgeConnectId                = "id"

	AnnotationEdgeConnectContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + EdgeConnectContainerName
)
