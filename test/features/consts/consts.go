package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"

	DevRegistryPullSecretName = "devregistry"
	EecImageRepo              = "public.ecr.aws/dynatrace/dynatrace-eec"
	EecImageTag               = "1.319.26.20250711-102845"
	LogMonitoringImageRepo    = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageTag     = "1.309.59.20250319-140247"
)
