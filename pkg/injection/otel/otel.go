package otel

import (
	"context"

	dtotel "github.com/Dynatrace/dynatrace-operator/pkg/util/otel"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, title string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(webhookotel.WebhookPodNameKey, webhookotel.GetWebhookPodName())))

	return dtotel.StartSpan(ctx, webhookotel.Tracer(), title, options...)
}
