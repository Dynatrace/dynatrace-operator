package extension

import (
	"context"

	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client       client.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider

	dynakube *dynatracev1beta3.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta3.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta3.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:       clt,
		apiReader:    apiReader,
		dynakube:     dynakube,
		timeProvider: timeprovider.New(),
	}
}

func (r *reconciler) Reconcile(_ context.Context) error {
	if r.dynakube.PrometheusEnabled() {
		log.Info("reconcile extensions")

		return nil
	}

	return nil
}
