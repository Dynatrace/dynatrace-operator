package otel

import (
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"golang.org/x/net/context"
)

func setupMetrics(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (*sdkmetric.MeterProvider, error) {
	meterExporter, err := newOtlpMetricsExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, err
	}
	return installMeterProvider(meterExporter, resource), nil
}

func installMeterProvider(exporter sdkmetric.Exporter, resource *resource.Resource) *sdkmetric.MeterProvider {
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(otelMetricReadInterval))),
	)

	otel.SetMeterProvider(meterProvider)
	log.Info("OTel meter provider installed successfully.")
	return meterProvider
}

func newOtlpMetricsExporter(ctx context.Context, endpoint string, apiToken string) (sdkmetric.Exporter, error) {
	// reset measurements after each cycle (i.e. after sending metrics to collector)
	var deltaTemporalitySelector = func(sdkmetric.InstrumentKind) metricdata.Temporality { return metricdata.DeltaTemporality }

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithTemporalitySelector(deltaTemporalitySelector),
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithURLPath(otelBaseApiUrl+otelMetricsUrl),
		otlpmetrichttp.WithHeaders(map[string]string{
			"Authorization": "Api-Token " + apiToken,
		}))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return exporter, nil
}
