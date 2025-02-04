package activegate

import "context"

type CapabilityReconciler interface {
	Reconcile(ctx context.Context) error
}
