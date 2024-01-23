package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "injection"
)

var tracer trace.Tracer

var once = sync.Once{}

func Tracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer(otelInstrumentationScope)
	})
	return tracer
}
