package proxy

const (
	proxyHostField     = "host"
	proxyPortField     = "port"
	proxyUsernameField = "username"
	proxyPasswordField = "password"

	ProxySecretMountPath  = "/var/lib/dynatrace/secrets/internal-proxy"
	ProxySecretVolumeName = "internal-proxy-secret-volume"

	ProxySecretHost          = "host"
	ProxySecretHostMountPath = ProxySecretMountPath + "/" + ProxySecretHost

	ProxySecretPort          = "port"
	ProxySecretPortMountPath = ProxySecretMountPath + "/" + ProxySecretPort

	ProxySecretUsername          = "username"
	ProxySecretUsernameMountPath = ProxySecretMountPath + "/" + ProxySecretUsername

	ProxySecretPassword          = "password"
	ProxySecretPasswordMountPath = ProxySecretMountPath + "/" + ProxySecretPassword
)
