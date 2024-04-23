package otel

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"os"
	"sync"
)

var envPodName string
var oncePodName = sync.Once{}

func GetCSIPodName() string {
	oncePodName.Do(func() {
		envPodName = os.Getenv("POD_NAME")
	})

	return envPodName
}

func SpanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(CsiPodNameKey, GetCSIPodName())))

	return options
}
