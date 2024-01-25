package proxy

const (
	hostField     = "host"
	portField     = "port"
	usernameField = "username"
	passwordField = "password"

	SecretMountPath  = "/var/lib/dynatrace/secrets/internal-proxy"
	SecretVolumeName = "internal-proxy-secret-volume"

	SecretHost          = "host"
	SecretHostMountPath = SecretMountPath + "/" + SecretHost

	SecretPort          = "port"
	SecretPortMountPath = SecretMountPath + "/" + SecretPort

	SecretUsername          = "username"
	SecretUsernameMountPath = SecretMountPath + "/" + SecretUsername

	SecretPassword          = "password"
	SecretPasswordMountPath = SecretMountPath + "/" + SecretPassword
)
