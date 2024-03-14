package dataingest_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	workloadEnrichmentVolumeName = "data-ingest-enrichment"
	ingestEndpointVolumeName     = "data-ingest-endpoint"
	enrichmentEndpointPath       = "/var/lib/dynatrace/enrichment/endpoint"
)

var (
	log = logd.Get().WithName("dataingest-pod-mutation")
)
