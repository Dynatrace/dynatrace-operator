package otel

import (
	"context"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func setupTraces(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (trace.TracerProvider, shutdownFn, error) {
	if !shouldUseOtel() {
		noopTracerProvider := trace.NewNoopTracerProvider()
		otel.SetTracerProvider(noopTracerProvider)
		log.Info("OTel noop tracer provider installed")
		return noopTracerProvider, noopShutdownFn, nil
	}

	tracerExporter, err := newOtlpTraceExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, noopShutdownFn, err
	}
	sdkTracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracerExporter),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(sdkTracerProvider)
	log.Info("OTel tracer provider installed successfully.")

	return sdkTracerProvider, sdkTracerProvider.Shutdown, nil
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

func StartSpan(ctx context.Context, tracer trace.Tracer, title string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil || title == "" {
		return ctx, noopSpan{}
	}
	return tracer.Start(ctx, title, opts...)
}

type noopSpan struct {
	trace.Span
}

var _ trace.Span = noopSpan{}

func (noopSpan) End(...trace.SpanEndOption) {}
