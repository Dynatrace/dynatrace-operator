package oneagent

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func handlePodListError(err error, listOps []client.ListOption) {
	log.Error(err, "failed to list pods", "listops", listOps)
}
