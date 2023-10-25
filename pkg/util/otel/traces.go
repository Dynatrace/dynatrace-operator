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

func setupTracesWithOtlp(ctx context.Context, resource *resource.Resource, endpoint string, apiToken string) (trace.TracerProvider, shutdownFn, error) {
	tracerExporter, err := newOtlpTraceExporter(ctx, endpoint, apiToken)
	if err != nil {
		return nil, nil, err
	}
	sdkTracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tracerExporter),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(sdkTracerProvider)
	log.Info("OpenTelementry tracer provider installed successfully.")

	return sdkTracerProvider, sdkTracerProvider.Shutdown, nil
}

func newOtlpTraceExporter(ctx context.Context, endpoint string, apiToken string) (sdktrace.SpanExporter, error) {
	if endpoint == "" || apiToken == "" {
		return nil, errors.Errorf("no endpoint or apiToken provided for OTLP traces exporter")
	}

	log.Info("setup OpenTelementry ingest", "endpoint", endpoint)
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

func StartSpan[T any](ctx context.Context, tracer T, title string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	var realTracer trace.Tracer
	switch t := any(tracer).(type) {
	case string:
		realTracer = otel.Tracer(t)
	case trace.Tracer:
		realTracer = t
	}

	if realTracer == nil || title == "" {
		return ctx, noopSpan{}
	}
	return realTracer.Start(ctx, title, opts...)
}

type noopSpan struct {
	trace.Span
}

var _ trace.Span = noopSpan{}

func (noopSpan) End(...trace.SpanEndOption) {}
