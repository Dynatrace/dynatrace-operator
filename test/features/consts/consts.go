package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"

	DevRegistryPullSecretName = "devregistry"
	EecImageRepo              = "478983378254.dkr.ecr.us-east-1.amazonaws.com/dynatrace/dynatrace-eec"
	EecImageTag               = "1.303.0.20240930-183404"
	LogMonitoringImageRepo    = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageTag     = "1.309.59.20250319-140247"
)
