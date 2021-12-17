package ingestendpoint

import "github.com/Dynatrace/dynatrace-operator/src/logger"

const (
	// SecretEndpointName is the name of the secret where the Operator replicates data-ingest data (data-ingest url, data-ingest token).
	SecretEndpointName = "dynatrace-data-ingest-endpoint"
)

var (
	log = logger.NewDTLogger().WithName("ingestendpoint")
)
