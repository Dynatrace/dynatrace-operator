package consts

const (
	// secret
	EecTokenSecretKey         = "eec.token"
	EecTokenSecretValuePrefix = "EEC dt0x01"

	OtelcTokenSecretKey         = "otelc.token"
	OtelcTokenSecretValuePrefix = "dt0x01"

	SecretSuffix = "-extensions-token"

	// shared volume name between eec and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsSecretConditionType  = "ExtensionsSecret"
	ExtensionsServiceConditionType = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"

	ExtensionsTlsSecretSuffix = "-extensions-controller-tls"

	// TLSKeyDataName is the key used to store a TLS private key in the secret's data field.
	TLSKeyDataName = "tls.key"

	// TLSCrtDataName is the key used to store a TLS certificate in the secret's data field.
	TLSCrtDataName = "tls.crt"
)
