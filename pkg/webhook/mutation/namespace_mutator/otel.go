package namespace_mutator

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// the attribute key needs to be added to the allow list on the receiving tenant
	mutatedNamespaceNameKey = "webhook.mutationrequest.namespace.name"
)

func countHandleMutationRequest(ctx context.Context, namespace string) {
	dtotel.Count(ctx, webhookotel.Meter(), "handledPodMutationRequests", int64(1),
		attribute.String(webhookotel.WebhookPodNameKey, webhookotel.GetWebhookPodName()),
		attribute.String(mutatedNamespaceNameKey, namespace))
}

func spanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(webhookotel.WebhookPodNameKey, webhookotel.GetWebhookPodName()),

		// TODO: this is just for showcasing now, should be removed in the future
		attribute.String("debug.info", "foobar")))
	return options
}
