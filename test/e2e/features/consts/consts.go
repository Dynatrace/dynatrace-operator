package consts

const (
	AgCertificate                   = "custom-cas/agcrt.pem"
	AgCertificateAndPrivateKey      = "custom-cas/agcrtkey.p12"
	AgCertificateAndPrivateKeyField = "server.p12"
	AgSecretName                    = "ag-ca"
	TelemetryIngestTLSSecretName    = "telemetry-ingest-tls"
	DevRegistryPullSecretName       = "devregistry"

	DefaultEecImage           = "public.ecr.aws/dynatrace/dynatrace-eec:1.327.30.20251107-111521"
	EecImageEnvVar            = "E2E_EEC_IMAGE"
	DefaultLogMonitoringImage = "public.ecr.aws/dynatrace/dynatrace-logmodule:1.309.59.20250319-140247"
	LogMonitoringImageEnvVar  = "E2E_LOGMON_IMAGE"
	DefaultKSPMImage          = "public.ecr.aws/dynatrace/dynatrace-k8s-node-config-collector:1.5.2"
	KSPMImageEnvVar           = "E2E_KSPM_IMAGE"
	DefaultOtelCollectorImage = "public.ecr.aws/dynatrace/dynatrace-otel-collector:latest"
	OtelCollectorImageEnvVar  = "E2E_OTELC_IMAGE"
	DefaultDBExecutorImage    = "public.ecr.aws/dynatrace/dynatrace-database-datasource-executor:1.327.41.20251114-153023"
	DBExecutorImageEnvVar     = "E2E_DB_EXECUTOR_IMAGE"
	DefaultCodeModulesImage   = "ghcr.io/dynatrace/dynatrace-bootstrapper:snapshot"
	CodeModulesImageEnvVar    = "E2E_CODEMODULES_IMAGE"
)
