package kspm

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm/token"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	tokenReconciler     *token.Reconciler
	daemonSetReconciler *daemonset.Reconciler
}

func NewReconciler(client client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		tokenReconciler:     token.NewReconciler(client, apiReader),
		daemonSetReconciler: daemonset.NewReconciler(client, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	err := r.tokenReconciler.Reconcile(ctx, dk)
	if err != nil {
		log.Info("failed to reconcile Dynatrace KSPM Secret")

		return err
	}

	err = r.daemonSetReconciler.Reconcile(ctx, dk)
	if err != nil {
		log.Info("failed to reconcile Dynatrace KSPM DaemonSet")

		return err
	}

	return nil
}
