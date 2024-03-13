package dtotel

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
)

const (
	otelSecretName     = "dynatrace-operator-otel-config"
	otelApiEndpointKey = "endpoint"
	otelAccessTokenKey = "apiToken"
	otelBaseApiUrl     = "/api/v2/otlp"

	otelMetricReadInterval         = 5 * time.Second
	golangRuntimeStatsReadInterval = otelMetricReadInterval

	// these are the default values defined by OpenTelementry spec but not exported as consts in SDK packages
	otelTracesUrl  = "/v1/traces"
	otelMetricsUrl = "/v1/metrics"
)

var log = logd.Get().WithName("open-telemetry")
