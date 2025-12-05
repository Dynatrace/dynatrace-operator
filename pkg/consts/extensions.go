package consts

const (
	ExtensionsSelfSignedTLSSecretSuffix = "-extension-controller-tls"

	// shared volume name between EEC and OtelC
	ExtensionsTokensVolumeName = "tokens"

	ExtensionsControllerSuffix         = "-extension-controller"
	ExtensionsDatasourceTargetPortName = "collector-com"
	ExtensionsDatasourceTargetPort     = 14599

	DatasourceTokenSecretKey         = "datasource.token"
	DatasourceTokenSecretValuePrefix = "dt0x01"

	// DatasourceLabelKey should be placed on all datasource deployments to allow EEC to separate them.
	DatasourceLabelKey = "extensions.dynatrace.com/datasource"
	// DatabaseDatasourceLabelValue must always be used for database extensions.
	DatabaseDatasourceLabelValue = "sql"
)
