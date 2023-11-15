package pod_mutator

import (
	"context"
	"os"
	"sync"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// the attribute key needs to be added to the allow list on the receiving tenant
	mutatedPodNameKey = "webhook.mutationrequest.pod.name"
	webhookPodNameKey = "k8s.pod.name"
)

var envPodName string
var once = sync.Once{}

func getWebhookPodName() string {
	once.Do(func() {
		envPodName = os.Getenv("POD_NAME")
	})
	return envPodName
}

func countHandleMutationRequest(ctx context.Context, mutatedPodName string) {
	dtotel.Count(ctx, webhookotel.Meter(), "handledPodMutationRequests", int64(1),
		attribute.String(webhookPodNameKey, getWebhookPodName()),
		attribute.String(mutatedPodNameKey, mutatedPodName))
}

func spanOptions(opts ...trace.SpanStartOption) []trace.SpanStartOption {
	options := make([]trace.SpanStartOption, 0)
	options = append(options, opts...)
	options = append(options, trace.WithAttributes(
		attribute.String(webhookPodNameKey, getWebhookPodName()),

		// TODO: this is just for showcasing now, should be removed in the future
		attribute.String("debug.info", "foobar")))
	return options
}
