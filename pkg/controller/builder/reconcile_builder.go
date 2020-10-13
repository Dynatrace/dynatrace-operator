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

func ReconcileImmediately() reconcile.Result {
	return ReconcileAfter(1 * time.Nanosecond)
}

const (
	FiveMinutes = 5 * time.Minute
)
