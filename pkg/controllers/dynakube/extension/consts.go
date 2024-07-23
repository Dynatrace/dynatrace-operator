package extension

const (
	// secret
	EecTokenSecretKey         = "eec-token"
	eecTokenSecretValuePrefix = "EEC dt0x01"
	secretSuffix              = "-extensions-token"

	extensionsTokenSecretConditionType = "ExtensionsTokenSecret"
	extensionsServiceConditionType     = "ExtensionsService"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorComPort        = 14599
	ExtensionsCollectorTargetPortName = "collector-com"
)
