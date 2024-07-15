package pod

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// the attribute key needs to be added to the allow list on the receiving tenant.
	mutatedPodNameKey            = "webhook.mutationrequest.pod.name"
	podMutationHandledMetricName = "handledPodMutationRequests"
)

var podName string

func init() {
	podName = os.Getenv("POD_NAME")
}

func countHandleMutationRequest(ctx context.Context, mutatedPodName string) {
	dtotel.Count(ctx, webhookotel.Meter, podMutationHandledMetricName, int64(1),
		attribute.String(webhookotel.WebhookPodNameKey, podName),
		attribute.String(mutatedPodNameKey, mutatedPodName))
}

func spanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(webhookotel.WebhookPodNameKey, podName)))

	return options
}
