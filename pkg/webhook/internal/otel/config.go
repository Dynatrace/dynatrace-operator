package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelInstrumentationScope = "webhook"
	WebhookPodNameKey        = "k8s.pod.name"
)

var Meter metric.Meter
var Tracer trace.Tracer

func init() {
	Tracer = otel.Tracer(otelInstrumentationScope)
	Meter = otel.Meter(otelInstrumentationScope)
}
