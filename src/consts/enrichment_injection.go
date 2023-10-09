package consts

const (
	EnrichmentEndpointSecretName = "dynatrace-data-ingest-endpoint"
	EnrichmentFilenameTemplate   = "dt_metadata.%s"
	EnrichmentInjectedEnv        = "DATA_INGEST_INJECTED"
	EnrichmentWorkloadKindEnv    = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv    = "DT_WORKLOAD_NAME"
	EnrichmentUnknownWorkload    = "UNKNOWN"
)

var (
	EnrichmentMountPath = "/var/lib/dynatrace/enrichment"
)
