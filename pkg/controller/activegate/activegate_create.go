package activegate

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileActiveGate) createPod(pod *corev1.Pod) (reconcile.Result, error) {
	log.Info("creating new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	err := r.client.Create(context.TODO(), pod)
	if err != nil {
		return reconcile.Result{}, err
	}
	// Sleep until pod is ready
	time.Sleep(TimeUntilActive)

	return builder.ReconcileAfterFiveMinutes(), nil
}
