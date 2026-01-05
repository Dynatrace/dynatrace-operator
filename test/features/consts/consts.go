package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"
	DevRegistryPullSecretName       = "devregistry"

	EecImageRepo           = "public.ecr.aws/dynatrace/dynatrace-eec"
	EecImageTag            = "1.327.30.20251107-111521"
	LogMonitoringImageRepo = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageTag  = "1.309.59.20250319-140247"
	KSPMImageRepo          = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	KSPMImageTag           = "1.5.2"
	OtelCollectorImageRepo = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	OtelCollectorImageTag  = "latest"
	DBExecutorImageRepo    = "public.ecr.aws/dynatrace/dynatrace-database-datasource-executor"
	DBExecutorImageTag     = "1.327.41.20251114-153023"
)
