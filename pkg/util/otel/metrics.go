package otel

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"golang.org/x/net/context"
)

func setupMetricsWithOtlp(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (metric.MeterProvider, shutdownFn, error) {
	meterExporter, err := newOtlpMetricsExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, nil, err
	}
	sdkMeterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(meterExporter, sdkmetric.WithInterval(otelMetricReadInterval))),
	)

	otel.SetMeterProvider(sdkMeterProvider)
	log.Info("OpenTelementry meter provider installed successfully.")

	return sdkMeterProvider, sdkMeterProvider.Shutdown, nil
}

func newOtlpMetricsExporter(ctx context.Context, endpoint string, apiToken string) (sdkmetric.Exporter, error) {
	if endpoint == "" || apiToken == "" {
		return nil, errors.Errorf("no endpoint or apiToken provided for OTLP metrics exporter")
	}

	// reset measurements after each cycle (i.e. after sending metrics to collector)
	deltaTemporalitySelector := func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }

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

// Count is a utility that can be used to increase Int64 and Float64 counters, but with same safeguards to avoid panics, if meter is not
// properly initialized
func Count[N Number](ctx context.Context, meter metric.Meter, name string, value N, attributes ...attribute.KeyValue) {
	if meter == nil || name == "" {
		return
	}

	var err error
	switch v := any(value).(type) {
	case int64:
		var counter metric.Int64Counter
		counter, err = meter.Int64Counter(name)
		if err == nil {
			counter.Add(ctx, v, metric.WithAttributes(attributes...))
		}
	case float64:
		var counter metric.Float64Counter
		counter, err = meter.Float64Counter(name)
		if err == nil {
			counter.Add(ctx, v, metric.WithAttributes(attributes...))
		}
	default:
		err = errors.Errorf("unsupported counter type")
	}
	if err != nil {
		log.Error(err, "failed counting", "metric", name)
	}
}
