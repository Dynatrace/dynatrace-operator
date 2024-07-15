package kubesystem

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "kubesystem"
)

var Tracer trace.Tracer

func init() {
	Tracer = otel.Tracer(otelInstrumentationScope)
}
