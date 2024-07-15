package otel

import (
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var envPodName string

func init() {
	envPodName = os.Getenv("POD_NAME")
}

func SpanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(CsiPodNameKey, envPodName)))

	return options
}
