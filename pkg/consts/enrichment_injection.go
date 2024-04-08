package consts

const (
	EnrichmentEndpointSecretName = "dynatrace-metadata-enrichment-endpoint"
	EnrichmentFilenameTemplate   = "dt_metadata.%s"
	EnrichmentInjectedEnv        = "METADATA_ENRICHMENT_INJECTED"
	EnrichmentWorkloadKindEnv    = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv    = "DT_WORKLOAD_NAME"
	EnrichmentUnknownWorkload    = "UNKNOWN"
)

var (
	EnrichmentMountPath = "/var/lib/dynatrace/enrichment"
)
