package consts

const (
	TelemetryApiCredentialsSecretName = "dynatrace-telemetry-api-credentials"

	ConfigFieldName                   = "telemetry.yaml"
	TelemetryCollectorConfigmapSuffix = "-telemetry-collector-config"

	CustomTlsCertMountPath = "/tls/custom/telemetry"

	EnvDataIngestToken = "DT_DATA_INGEST_TOKEN"
)
