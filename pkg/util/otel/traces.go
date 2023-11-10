package otel

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"path/filepath"
	"runtime"
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

func StartSpan[T any](ctx context.Context, tracer T, _ string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	caller := getCaller(1)

	realTracer := resolveTracer(tracer)
	if realTracer == nil {
		log.Info("failed to start span, no valid tracer given", "caller", caller, "tracer", fmt.Sprintf("%v", tracer))
		return ctx, noopSpan{}
	}

	return realTracer.Start(ctx, caller, opts...)
}

func resolveTracer[T any](tracer T) trace.Tracer {
	var realTracer trace.Tracer
	switch t := any(tracer).(type) {
	case string:
		realTracer = otel.Tracer(t)
	case trace.Tracer:
		realTracer = t
	}
	return realTracer
}

func getCaller(i int) string {
	if pc, filePath, line, ok := runtime.Caller(i); ok {
		details := runtime.FuncForPC(pc)
		fileName := filepath.Base(filePath)
		functionName := filepath.Base(details.Name())
		return fmt.Sprintf("%s (%s:%d)", functionName, fileName, line)
	}
	return "<unknown function>"
}

type noopSpan struct {
	trace.Span
}

var _ trace.Span = noopSpan{}

func (noopSpan) End(...trace.SpanEndOption) {}
