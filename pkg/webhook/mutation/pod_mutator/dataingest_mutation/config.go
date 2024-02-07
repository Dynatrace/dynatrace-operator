package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

const (
	workloadEnrichmentVolumeName = "data-ingest-enrichment"
	ingestEndpointVolumeName     = "data-ingest-endpoint"
	enrichmentEndpointPath       = "/var/lib/dynatrace/enrichment/endpoint"
)

var (
	log = logger.Get().WithName("dataingest-pod-mutation")
)
