package consts

const (
	EdgeConnectAnnotationSecretHash  = "my-secret-hash"
	EdgeConnectUserProvisioned       = "user-provisioned"
	EdgeConnectContainerName         = "edge-connect"
	EdgeConnectServiceAccountName    = "dynatrace-edgeconnect"
	EdgeConnectMountPath             = "/etc/ssl"
	EdgeConnectCustomCertificateName = "certificate.cer"
	EdgeConnectCustomCAVolumeName    = "ca-certs"
	EdgeConnectConfigFileName        = "edgeConnect.yaml"
	EdgeConnectConfigPath            = "/" + EdgeConnectConfigFileName
	EdgeConnectConfigVolumeMountName = "ec-vm"
	EdgeConnectSecretSuffix          = "ec-yaml"
	EdgeConnectCAConfigMapKey        = "certs"

	KeyEdgeConnectOauthClientID     = "oauth-client-id"
	KeyEdgeConnectOauthClientSecret = "oauth-client-secret"
	KeyEdgeConnectOauthResource     = "oauth-client-resource"
	KeyEdgeConnectId                = "id"

	AnnotationEdgeConnectContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + EdgeConnectContainerName
)
