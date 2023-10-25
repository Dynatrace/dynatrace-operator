package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	OpenTelemetryServiceName = "DynatraceWebhook"
	WebhookPodNameKey        = "k8s.pod.name"
)

var tracer trace.Tracer
var meter metric.Meter

var once = sync.Once{}

func Meter() metric.Meter {
	once.Do(func() {
		tracer = otel.Tracer(OpenTelemetryServiceName)
		meter = otel.Meter(OpenTelemetryServiceName)
	})
	return meter
}

func Tracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer(OpenTelemetryServiceName)
		meter = otel.Meter(OpenTelemetryServiceName)
	})
	return tracer
}
