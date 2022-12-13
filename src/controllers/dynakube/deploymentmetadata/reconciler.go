package deploymentmetadata

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	context   context.Context
	client    client.Client
	apiReader client.Reader
	dynakube  *dynatracev1beta1.DynaKube
}

func NewReconciler(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) *Reconciler { //nolint:revive // argument-limit doesn't apply to constructors
	return &Reconciler{
		context:   ctx,
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
	}
}

func (r *Reconciler) Reconcile() error {
	// TODO
	return nil
}

