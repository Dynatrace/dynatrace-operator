package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/eec"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider

	dk *dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dk:           dk,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	log.Info("start reconciling extensions")

	err := r.reconcileSecret(ctx)
	if err != nil {
		return err
	}

	err = r.reconcileService(ctx)
	if err != nil {
		return err
	}

	err = tls.NewReconciler(r.client, r.apiReader, r.dk).Reconcile(ctx)
	if err != nil {
		return err
	}

	err = eec.NewReconciler(r.client, r.apiReader, r.dk).Reconcile(ctx)
	if err != nil {
		return err
	}

	err = otel.NewReconciler(r.client, r.apiReader, r.dk).Reconcile(ctx)
	if err != nil {
		return err
	}

	return nil
}
