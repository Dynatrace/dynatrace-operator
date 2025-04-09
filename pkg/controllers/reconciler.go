package controllers

import "context"

type Reconciler interface {
	Reconcile(ctx context.Context) error
}
