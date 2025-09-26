package consts

const (
	ExtensionsSelfSignedTLSSecretSuffix = "-extensions-controller-tls"

	// shared volume name between eec and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsControllerSuffix        = "-extensions-controller"
	ExtensionsCollectorTargetPortName = "collector-com"

	DatasourceTokenSecretKey         = "datasource.token"
	DatasourceTokenSecretValuePrefix = "dt0x01"
)
