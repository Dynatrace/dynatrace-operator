package otel

import (
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func SpanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(webhookotel.WebhookPodNameKey, webhookotel.GetWebhookPodName())))
	return options
}
