package kubeobjects

type Reconciler interface {
	Reconcile() (update bool, err error)
}
