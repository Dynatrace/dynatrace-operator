package builder

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func ReconcileAfter(duration time.Duration) reconcile.Result {
	return reconcile.Result{RequeueAfter: duration}
}

func ReconcileAfterFiveMinutes() reconcile.Result {
	return ReconcileAfter(FiveMinutes)
}

const (
	FiveMinutes = 5 * time.Minute
)
