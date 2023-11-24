package kubesystem

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "kubesystem"
)

var kubeSystemTracer trace.Tracer
var once = sync.Once{}

func tracer() trace.Tracer {
	once.Do(func() {
		kubeSystemTracer = otel.Tracer(otelInstrumentationScope)
	})
	return kubeSystemTracer
}
