package consts

import "github.com/Dynatrace/dynatrace-operator/pkg/api"

const (
	EdgeConnectAnnotationSecretHash  = api.InternalFlagPrefix + "secret-hash"
	EdgeConnectUserProvisioned       = "user-provisioned"
	EdgeConnectContainerName         = "edge-connect"
	EdgeConnectMountPath             = "/etc/ssl"
	EdgeConnectCustomCertificateName = "certificate.cer"
	EdgeConnectCustomCAVolumeName    = "ca-certs"
	EdgeConnectConfigFileName        = "edgeConnect.yaml"
	EdgeConnectConfigPath            = "/" + EdgeConnectConfigFileName
	EdgeConnectConfigVolumeMountName = "ec-vm"
	EdgeConnectSecretSuffix          = "ec-yaml"
	EdgeConnectCAConfigMapKey        = "certs"
	EdgeConnectServiceAccountCAPath  = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	KeyEdgeConnectOauthClientID     = "oauth-client-id"
	KeyEdgeConnectOauthClientSecret = "oauth-client-secret"
	KeyEdgeConnectOauthResource     = "oauth-client-resource"
	KeyEdgeConnectId                = "id"

	AnnotationEdgeConnectContainerAppArmor = "container.apparmor.security.beta.kubernetes.io/" + EdgeConnectContainerName

	// SecretConfigConditionType identifies the secret config condition.
	SecretConfigConditionType = "SecretConfigConditionType"
)
