package otelc

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/statefulset"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client                client.Client
	apiReader             client.Reader
	dk                    *dynakube.DynaKube
	statefulsetReconciler controllers.Reconciler
}

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler { //nolint
	return &Reconciler{
		client:                client,
		apiReader:             apiReader,
		dk:                    dk,
		statefulsetReconciler: statefulset.NewReconciler(client, apiReader, dk),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	err := r.statefulsetReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("failed to reconcile Dynatrace OTELc statefulset")

		return err
	}

	return nil
}
