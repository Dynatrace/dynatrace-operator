package config

const (
	EnrichmentEndpointSecretName = "dynatrace-data-ingest-endpoint" //nolint:gosec
	EnrichmentFilenameTemplate   = "dt_metadata.%s"
	EnrichmentInjectedEnv        = "DATA_INGEST_INJECTED"
	EnrichmentWorkloadKindEnv    = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv    = "DT_WORKLOAD_NAME"
	EnrichmentUnknownWorkload    = "UNKNOWN"
)

var (
	EnrichmentMountPath = "/var/lib/dynatrace/enrichment"
)
