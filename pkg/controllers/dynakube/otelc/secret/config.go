package secret

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("telemetry-service-api-credentials-secret")
)
