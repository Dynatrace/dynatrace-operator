package otel

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"golang.org/x/net/context"
)

func setupMetrics(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (metric.MeterProvider, shutdownFn, error) {
	if !shouldUseOtel() {
		// the Noop implementation is guaranteed to do nothing and keep the impact low
		noopMeterProvider := noop.NewMeterProvider()
		otel.SetMeterProvider(noopMeterProvider)
		log.Info("OTel noop meter provider installed")
		return noopMeterProvider, noopShutdownFn, nil
	}

	meterExporter, err := newOtlpMetricsExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, noopShutdownFn, err
	}
	sdkMeterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(meterExporter, sdkmetric.WithInterval(otelMetricReadInterval))),
	)

	otel.SetMeterProvider(sdkMeterProvider)
	log.Info("OTel meter provider installed successfully.")

	return sdkMeterProvider, sdkMeterProvider.Shutdown, nil
}

func newOtlpMetricsExporter(ctx context.Context, endpoint string, apiToken string) (sdkmetric.Exporter, error) {
	// reset measurements after each cycle (i.e. after sending metrics to collector)
	var deltaTemporalitySelector = func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }

	aggregationSelector := func(ik sdkmetric.InstrumentKind) sdkmetric.Aggregation {
		switch ik {
		case sdkmetric.InstrumentKindHistogram:
			// Dynatrace doesn't accept histograms yet, lets drop them
			return sdkmetric.AggregationDrop{}
		}
		return sdkmetric.DefaultAggregationSelector(ik)
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithTemporalitySelector(deltaTemporalitySelector),
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithURLPath(otelBaseApiUrl+otelMetricsUrl),
		otlpmetrichttp.WithHeaders(map[string]string{
			"Authorization": "Api-Token " + apiToken,
		}),
		otlpmetrichttp.WithAggregationSelector(aggregationSelector))

	if err != nil {
		return nil, errors.WithStack(err)
	}
	return exporter, nil
}
