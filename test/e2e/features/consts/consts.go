package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"
	DevRegistryPullSecretName       = "devregistry"

	DefaultEECRepo           = "public.ecr.aws/dynatrace/dynatrace-eec"
	EECImageEnvVar           = "E2E_EEC_IMAGE"
	DefaultLogMonitoringRepo = "public.ecr.aws/dynatrace/dynatrace-logmodule"
	LogMonitoringImageEnvVar = "E2E_LOGMON_IMAGE"
	DefaultKSPMRepo          = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector"
	KSPMImageEnvVar          = "E2E_KSPM_IMAGE"
	DefaultOtelCollectorRepo = "public.ecr.aws/dynatrace/dynatrace-otel-collector"
	OtelCollectorImageEnvVar = "E2E_OTELC_IMAGE"
	DefaultDBExecutorRepo    = "public.ecr.aws/dynatrace/dynatrace-database-datasource-executor"
	DBExecutorImageEnvVar    = "E2E_DB_EXECUTOR_IMAGE"
	DefaultCodeModulesImage  = "ghcr.io/dynatrace/dynatrace-bootstrapper:snapshot"
	CodeModulesImageEnvVar   = "E2E_CODEMODULES_IMAGE"
)
