package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "csi"
	CsiPodNameKey            = "k8s.pod.name"
)

var tracer trace.Tracer
var meter metric.Meter //nolint

var once = sync.Once{}

func Tracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer(otelInstrumentationScope)
		meter = otel.Meter(otelInstrumentationScope)
	})

	return tracer
}
