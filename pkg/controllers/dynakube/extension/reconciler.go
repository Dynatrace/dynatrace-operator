package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/databases"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/eec"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client       client.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:       clt,
		apiReader:    apiReader,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("start reconciling extensions")

	err := r.reconcileSecret(ctx, dk)
	if err != nil {
		return err
	}

	err = r.reconcileService(ctx, dk)
	if err != nil {
		return err
	}

	err = tls.NewReconciler(r.client, r.apiReader, dk).Reconcile(ctx)
	if err != nil {
		return err
	}

	err = eec.NewReconciler(r.client, r.apiReader, dk).Reconcile(ctx)
	if err != nil {
		return err
	}

	if err := databases.NewReconciler(r.client, r.apiReader, dk).Reconcile(ctx); err != nil {
		return err
	}

	return nil
}
