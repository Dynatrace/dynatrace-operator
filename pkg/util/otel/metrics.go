package otel

import (
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
		if ik == sdkmetric.InstrumentKindHistogram {
			// Dynatrace doesn't ingest histograms yet, drop it
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

type Number interface {
	int64 | float64
}

func Count[N Number](ctx context.Context, meter metric.Meter, name string, value N, attributes ...any) {
	if meter == nil || name == "" {
		return
	}

	metricAttributes := make([]attribute.KeyValue, 0)
	for i := 0; i < len(attributes)-1; i += 2 {
		// if the number of attributes is uneven, the last entry is considered a key without value and will be dropped
		metricAttributes = append(metricAttributes, attribute.KeyValue{
			Key: attribute.Key(fmt.Sprintf("%s", attributes[i])),
			// converting all values to strings here for simplicity, could be more sophisticated with a giant type switch
			Value: attribute.StringValue(fmt.Sprintf("%s", attributes[i+1])),
		})
	}

	var err error
	switch v := any(value).(type) {
	case int64:
		var counter metric.Int64Counter
		counter, err = meter.Int64Counter(name)
		if err == nil {
			counter.Add(ctx, v, metric.WithAttributes(metricAttributes...))
		}
	case float64:
		var counter metric.Float64Counter
		counter, err = meter.Float64Counter(name)
		if err == nil {
			counter.Add(ctx, v, metric.WithAttributes(metricAttributes...))
		}
	}
	if err != nil {
		log.Error(err, "failed counting", "metric", name)
	}
}
