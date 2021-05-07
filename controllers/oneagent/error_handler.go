package oneagent

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func handlePodListError(logger logr.Logger, err error, listOps []client.ListOption) {
	logger.Error(err, "failed to list pods", "listops", listOps)
}
