package kubeobjects

type PseudoReconciler interface {
	Reconcile() (update bool, err error)
}
