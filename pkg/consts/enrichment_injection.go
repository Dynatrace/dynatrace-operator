package consts

const (
	EnrichmentEndpointSecretName = "dynatrace-metadata-enrichment-endpoint"
	EnrichmentInjectedEnv        = "METADATA_ENRICHMENT_INJECTED"
	EnrichmentWorkloadKindEnv    = "K8S_WORKLOAD_KIND"
	EnrichmentWorkloadNameEnv    = "K8S_WORKLOAD_NAME"
	EnrichmentClusterNameEnv     = "K8S_CLUSTER_NAME"
	EnrichmentPropertiesFilename = "dt_metadata.properties"
	EnrichmentJsonFilename       = "dt_metadata.json"
	EnrichmentMountPath          = "/var/lib/dynatrace/enrichment"
	EnrichmentInitPath           = "/tmp/enrichment"
)
