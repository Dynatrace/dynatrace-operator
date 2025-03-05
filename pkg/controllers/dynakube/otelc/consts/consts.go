package consts

const (
	TelemetryApiCredentialsSecretName = "dynatrace-telemetry-api-credentials"

	ConfigFieldName                   = "telemetry.yaml"
	TelemetryCollectorConfigmapSuffix = "-telemetry-collector-config"

	CustomTlsCertMountPath             = "/tls/custom/telemetry"
	TrustedCAFilename                  = "certs"
	TrustedCAVolumeMountPath           = "/tls/custom/cacerts"
	TrustedCAVolumePath                = TrustedCAVolumeMountPath + "/" + TrustedCAFilename
	ActiveGateTLSCertCAVolumeMountPath = "/tls/custom/activegate"
	ActiveGateTLSCertVolumePath        = ActiveGateTLSCertCAVolumeMountPath + "/" + TrustedCAFilename

	EnvDataIngestToken = "DT_DATA_INGEST_TOKEN"
)
