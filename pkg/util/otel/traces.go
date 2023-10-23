package otel

import (
	"context"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func setupTraces(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (*sdktrace.TracerProvider, error) {
	tracerExporter, err := newOtlpTraceExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, err
	}

	return installTraceProvider(tracerExporter, resource), nil
}

func installTraceProvider(exporter sdktrace.SpanExporter, resource *resource.Resource) *sdktrace.TracerProvider {
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	otel.SetTracerProvider(tracerProvider)

	log.Info("OTel tracer provider installed successfully.")
	return tracerProvider
}

func newOtlpTraceExporter(ctx context.Context, endpoint string, apiToken string) (sdktrace.SpanExporter, error) {
	log.Info("setup OTel ingest", "endpoint", endpoint)
	otlpHttpClient := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath(otelBaseApiUrl+otelTracesUrl),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Api-Token " + apiToken,
		}))
	exporter, err := otlptrace.New(ctx, otlpHttpClient)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return exporter, nil
}
