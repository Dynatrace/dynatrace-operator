package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "webhook"
	WebhookPodNameKey        = "k8s.pod.name"
)

var tracer trace.Tracer
var meter metric.Meter

var once = sync.Once{}

func Meter() metric.Meter {
	once.Do(func() {
		tracer = otel.Tracer(otelInstrumentationScope)
		meter = otel.Meter(otelInstrumentationScope)
	})
	return meter
}

func Tracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer(otelInstrumentationScope)
		meter = otel.Meter(otelInstrumentationScope)
	})
	return tracer
}
