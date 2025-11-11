package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"
	DevRegistryPullSecretName       = "devregistry"

	EecImageRepo           = "public.ecr.aws/dynatrace/dynatrace-eec"
	EecImageTag            = "1.319.26.20250711-102845"
	LogMonitoringImageRepo = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageTag  = "1.309.59.20250319-140247"
	KSPMImageRepo          = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	KSPMImageTag           = "1.5.2"
	OtelCollectorImageRepo = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	OtelCollectorImageTag  = "latest"
	DBExecutorImageRepo    = "478983378254.dkr.ecr.us-east-1.amazonaws.com/dynatrace/dynatrace-database-datasource-executor"
	DBExecutorImageTag     = "1.329.0.20251106-113915"
)
