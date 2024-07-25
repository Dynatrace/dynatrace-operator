package extension

const (
	EecTokenSecretKey         = "eec-token"
	eecTokenSecretValuePrefix = "EEC dt0x01"

	otelcTokenSecretKey         = "otelc-token"
	otelcTokenSecretValuePrefix = "dt0x01"

	secretSuffix = "-extensions-token"

	extensionsSecretConditionType  = "ExtensionsSecret"
	extensionsServiceConditionType = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"
)
