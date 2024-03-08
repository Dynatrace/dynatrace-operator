package dtotel

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan[T any](ctx context.Context, tracer T, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	const spanTitleCallerDepth = 2
	spanTitle := getCaller(spanTitleCallerDepth)

	realTracer := resolveTracer(tracer)
	if realTracer == nil {
		log.Info("failed to start span, no valid tracer given", "spanTitle", spanTitle, "tracer", fmt.Sprintf("%v", tracer))

		return ctx, noopSpan{spanTitle: spanTitle}
	}

	return realTracer.Start(ctx, spanTitle, opts...)
}

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
	spanTitle string
}

var _ trace.Span = noopSpan{}

func (noopSpan) End(...trace.SpanEndOption) {}

func (s noopSpan) String() string {
	return s.spanTitle
}
