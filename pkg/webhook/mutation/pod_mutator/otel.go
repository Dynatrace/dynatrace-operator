package pod_mutator

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/otel"
)

func (webhook *podMutatorWebhook) countHandleMutationRequest(ctx context.Context) {
	envPodName := os.Getenv("POD_NAME")
	otel.Count(ctx, webhook.otelMeter, "handledPodMutationRequests", int64(1), "podName", envPodName)
}
