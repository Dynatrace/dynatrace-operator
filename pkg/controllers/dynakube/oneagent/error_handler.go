package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func handlePodListError(ctx context.Context, err error, listOps []client.ListOption) {
	log := logd.FromContext(ctx)
	log.Error(err, "failed to list pods", "listops", listOps)
}
