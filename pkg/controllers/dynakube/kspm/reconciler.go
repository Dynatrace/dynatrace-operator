package kspm

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube,
) controllers.Reconciler

func NewReconciler( //nolint
	client client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube,
) controllers.Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
		dk:        dk,
	}
}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.IsKSPMEnabled() {
		return ensureKSPMToken(ctx, r.client, r.apiReader, r.dk)
	}

	return nil
}
