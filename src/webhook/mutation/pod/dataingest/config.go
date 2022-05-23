package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	EnrichmentVolumeName         = "data-ingest-enrichment"
	EnrichmentEndpointVolumeName = "data-ingest-endpoint"
	EnrichmentEndpointPath       = "/var/lib/dynatrace/enrichment/endpoint"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.dataIngest")
)
