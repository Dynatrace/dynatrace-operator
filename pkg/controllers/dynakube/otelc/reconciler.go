package otelc

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/configuration"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/endpoint"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/service"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/statefulset"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client                  client.Client
	apiReader               client.Reader
	statefulsetReconciler   *statefulset.Reconciler
	serviceReconciler       *service.Reconciler
	endpointReconciler      *endpoint.Reconciler
	configurationReconciler *configuration.Reconciler
}

func NewReconciler(client client.Client, apiReader client.Reader) *Reconciler { //nolint
	return &Reconciler{
		client:                  client,
		apiReader:               apiReader,
		statefulsetReconciler:   statefulset.NewReconciler(client, apiReader),
		serviceReconciler:       service.NewReconciler(client, apiReader),
		endpointReconciler:      endpoint.NewReconciler(client, apiReader),
		configurationReconciler: configuration.NewReconciler(client, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	err := r.serviceReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.endpointReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.configurationReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.statefulsetReconciler.Reconcile(ctx, dk)
	if err != nil {
		log.Info("failed to reconcile Dynatrace OTELc statefulset")

		return err
	}

	return nil
}
