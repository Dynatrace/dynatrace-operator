package proxy

const (
	schemeField   = "scheme"
	hostField     = "host"
	portField     = "port"
	usernameField = "username"
	passwordField = "password"

	SecretMountPath  = "/var/lib/dynatrace/secrets/internal-proxy"
	SecretVolumeName = "internal-proxy-secret-volume"
)
