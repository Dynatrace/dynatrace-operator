package activegate

import (
	"context"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *ReconcileActiveGate) deletePods(logger logr.Logger, pods []v1.Pod) error {
	for _, pod := range pods {
		logger.Info("deleting outdated pod", "pod", pod.Name, "node", pod.Spec.NodeName)
		err := r.client.Delete(context.TODO(), &pod)
		if err != nil && !errors.IsNotFound(err) {
			// Not an error if pod is already gone
			return err
		}
	}
	return nil
}
