package controllers

type Reconciler interface {
	Reconcile() error
}
