package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/metric"
)

func startAutoInstrumentation(meterProvider metric.MeterProvider) {
	// start go runtime auto instrumentation
	err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(golangRuntimeStatsReadInterval),
		runtime.WithMeterProvider(meterProvider))
	if err != nil {
		log.Error(err, "failed to start runtime instrumentation")
	}

	// start host auto instrumentation
	err = host.Start(host.WithMeterProvider(meterProvider))
	if err != nil {
		log.Error(err, "failed to start host instrumentation")
	}
}
