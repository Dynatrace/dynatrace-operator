package controller_runtime

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const otelInstrumentationScope = "controller_runtime"

var tracer trace.Tracer
var once = sync.Once{}

func controllerRuntimeTracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer(otelInstrumentationScope)
	})
	return tracer
}
