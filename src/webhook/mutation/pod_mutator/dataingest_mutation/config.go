package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	workloadEnrichmentVolumeName = "data-ingest-enrichment"
	ingestEndpointVolumeName     = "data-ingest-endpoint"
	enrichmentEndpointPath       = "/var/lib/dynatrace/enrichment/endpoint"
)

var (
	log = logger.NewDTLogger().WithName("pod.mutation-webhook.dataingest")
)
