package consts

const (
	OtlpAPIEndpointConfigMapName = "dynatrace-otlp-api-endpoint"

	ConfigFieldName                   = "telemetry.yaml"
	TelemetryCollectorConfigmapSuffix = "-telemetry-collector-config"

	CustomTLSCertMountPath = "/tls/custom/telemetry"

	TrustedCAsFile           = "rootca.pem"
	TrustedCAVolumeMountPath = "/tls/custom/cacerts"
	TrustedCAVolumePath      = TrustedCAVolumeMountPath + "/" + TrustedCAsFile

	ActiveGateCertFile                 = "cert.pem"
	ActiveGateTLSCertCAVolumeMountPath = "/tls/custom/activegate"
	ActiveGateTLSCertVolumePath        = ActiveGateTLSCertCAVolumeMountPath + "/" + ActiveGateCertFile

	EnvDataIngestToken = "DT_DATA_INGEST_TOKEN"
)
