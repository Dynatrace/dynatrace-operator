package controllers

import "golang.org/x/net/context"

type Reconciler interface {
	Reconcile(ctx context.Context) error
}
