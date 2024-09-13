package consts

const (
	EnrichmentEndpointSecretName     = "dynatrace-metadata-enrichment-endpoint"
	EnrichmentInjectedEnv            = "METADATA_ENRICHMENT_INJECTED"
	EnrichmentWorkloadKindEnv        = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv        = "DT_WORKLOAD_NAME"
	EnrichmentWorkloadAnnotationsEnv = "DT_WORKLOAD_ANNOTATIONS"
	EnrichmentClusterEntityIDEnv     = "DT_ENTITY_ID"
	EnrichmentClusterNameEnv         = "DT_CLUSTER_NAME"

	EnrichmentMountPath          = "/var/lib/dynatrace/enrichment"
	EnrichmentPropertiesFilename = "dt_metadata.properties"
	EnrichmentJsonFilename       = "dt_metadata.json"

	EnrichmentInitPath                       = "/tmp/enrichment"
	EnrichmentInitPropertiesFilenameTemplate = "dt_metadata_%s.properties"
	EnrichmentInitJsonFilenameTemplate       = "dt_metadata_%s.json"
)
