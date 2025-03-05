package consts

const (
	TelemetryApiCredentialsSecretName = "dynatrace-telemetry-api-credentials"

	ConfigFieldName                   = "telemetry.yaml"
	TelemetryCollectorConfigmapSuffix = "-telemetry-collector-config"

	CustomTlsCertMountPath   = "/tls/custom/telemetry"
	TrustedCAVolumeMountPath = "/tls/custom/cacerts"
	TrustedCAVolumePath      = TrustedCAVolumeMountPath + "/certs"

	EnvDataIngestToken = "DT_DATA_INGEST_TOKEN"
)
