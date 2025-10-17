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

	// DatasourceLabelKey should be placed on all datasource deployments to allow EEC to separate them.
	DatasourceLabelKey = "extensions.dynatrace.com/datasource"
	// DatabaseDatasourceLabelValue must always be used for database extensions.
	DatabaseDatasourceLabelValue = "sql"
)
