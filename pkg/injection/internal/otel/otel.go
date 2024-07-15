package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "injection"
)

var Tracer trace.Tracer

func init() {
	Tracer = otel.Tracer(otelInstrumentationScope)
}
