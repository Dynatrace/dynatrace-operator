package controller_runtime

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const otelInstrumentationScope = "controller_runtime"

var ControllerRuntimeTracer trace.Tracer

func init() {
	ControllerRuntimeTracer = otel.Tracer(otelInstrumentationScope)
}
