package consts

const (
	EnrichmentEndpointSecretName     = "dynatrace-metadata-enrichment-endpoint"
	EnrichmentEndpointFilename       = "endpoint.properties"
	EnrichmentInjectedEnv            = "METADATA_ENRICHMENT_INJECTED"
	EnrichmentWorkloadKindEnv        = "DT_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv        = "DT_WORKLOAD_NAME"
	EnrichmentWorkloadAnnotationsEnv = "DT_WORKLOAD_ANNOTATIONS"
	EnrichmentClusterEntityIDEnv     = "DT_ENTITY_ID"
	EnrichmentClusterNameEnv         = "DT_CLUSTER_NAME"

	EnrichmentSubDirName         = "enrichment"
	EnrichmentPropertiesFilename = "dt_metadata.properties"
	EnrichmentJsonFilename       = "dt_metadata.json"
	EnrichmentEndpointFilePath   = "endpoint/" + EnrichmentEndpointFilename
	EnrichmentEndpointVolumeName = "metadata-enrichment-endpoint"
	EnrichmentEndpointMountPath  = "/mnt/enrichment-endpoint"
)
