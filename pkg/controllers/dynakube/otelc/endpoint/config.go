package endpoint

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("telemetry-ingest-api-credentials-secret")
)
