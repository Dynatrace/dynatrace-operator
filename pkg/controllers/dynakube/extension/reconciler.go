// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/databases"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/eec"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type subReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

type imageAwareSubReconciler interface {
	Reconcile(ctx context.Context, imageClient dtimage.Client, dk *dynakube.DynaKube) error
}

type Reconciler struct {
	client              client.Client
	apiReader           client.Reader
	timeProvider        *timeprovider.Provider
	secrets             k8ssecret.QueryObject
	tlsReconciler       subReconciler
	eecReconciler       imageAwareSubReconciler
	databasesReconciler imageAwareSubReconciler
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:              clt,
		apiReader:           apiReader,
		timeProvider:        timeprovider.New(),
		secrets:             k8ssecret.Query(clt, apiReader),
		tlsReconciler:       tls.NewReconciler(clt, apiReader),
		eecReconciler:       eec.NewReconciler(clt, apiReader),
		databasesReconciler: databases.NewReconciler(clt, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, imageClient dtimage.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "extension")
	log.Info("start reconciling extensions")

	err := r.reconcileSecret(ctx, dk)
	if err != nil {
		return err
	}

	err = r.reconcileService(ctx, dk)
	if err != nil {
		return err
	}

	err = r.tlsReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	err = r.eecReconciler.Reconcile(ctx, imageClient, dk)
	if err != nil {
		return err
	}

	if err := r.databasesReconciler.Reconcile(ctx, imageClient, dk); err != nil {
		return err
	}

	return nil
}
