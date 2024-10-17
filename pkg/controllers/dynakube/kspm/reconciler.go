package kspm

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm/token"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client          client.Client
	apiReader       client.Reader
	dk              *dynakube.DynaKube
	tokenReconciler controllers.Reconciler
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler { //nolint
	return &Reconciler{
		client:          client,
		apiReader:       apiReader,
		dk:              dk,
		tokenReconciler: token.NewReconciler(client, apiReader, dk),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.tokenReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("failed to reconcile Dynatrace KSPM Secret")

		return err
	}

	return nil
}
