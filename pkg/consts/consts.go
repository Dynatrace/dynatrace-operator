package consts

const (
	OTELCollectorNameSuffix = "-otel-collector"
	NodeCollectorNameSuffix = "-node-config-collector"

	DTComponentsSecretsRootDir = "/var/lib/dynatrace/secrets"

	HostAvailabilityDetectionEnvVar = "DT_HOST_AVAILABILITY_DETECTION"

	OTLPExporterSecretName      = "dynatrace-otlp-exporter-config"
	OTLPExporterCertsSecretName = "dynatrace-otlp-exporter-certs"

	BootstrapperInitSecretName      = "dynatrace-bootstrapper-config"
	BootstrapperInitCertsSecretName = "dynatrace-bootstrapper-certs"

	AgentInitBinDirMount = "/mnt/bin"
)
