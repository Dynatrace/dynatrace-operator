package consts

const (
	EnrichmentEndpointSecretName = "dynatrace-metadata-enrichment-endpoint"
	EnrichmentInjectedEnv        = "METADATA_ENRICHMENT_INJECTED"
	EnrichmentWorkloadKindEnv    = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv    = "DT_WORKLOAD_NAME"
	EnrichmentPropertiesFilename = "dt_metadata.properties"
	EnrichmentJsonFilename       = "dt_metadata.json"
	EnrichmentMountPath          = "/var/lib/dynatrace/enrichment"
	EnrichmentInitPath           = "/tmp/enrichment"
)
