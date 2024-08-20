package consts

const (
	// secret
	EecTokenSecretKey         = "eec.token"
	EecTokenSecretValuePrefix = "EEC dt0x01"

	OtelcTokenSecretKey         = "otelc.token"
	OtelcTokenSecretValuePrefix = "dt0x01"

	SecretSuffix = "-extensions-token"

	ExtensionsSecretConditionType  = "ExtensionsSecret"
	ExtensionsServiceConditionType = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"

	ExtensionsCustomTlsCertificate = "custom-tls-certificates"
	ExtensionsTLScertFilename      = "tls.crt"
)
