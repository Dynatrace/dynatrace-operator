package consts

const (
	ExtensionsSelfSignedTLSSecretSuffix = "-extensions-controller-tls"

	// shared volume name between eec and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorTargetPortName = "collector-com"
	ExtensionsCollectorTargetPort     = 14599

	DatasourceTokenSecretKey         = "datasource.token"
	DatasourceTokenSecretValuePrefix = "dt0x01"
)
