package version

import (
	"context"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	edgeConnect  *edgeconnectv1alpha1.EdgeConnect
	timeProvider *timeprovider.Provider

	apiReader      client.Reader
	registryClient registry.ImageGetter
}

func NewReconciler(apiReader client.Reader, registryClient registry.ImageGetter, timeProvider *timeprovider.Provider, edgeConnect *edgeconnectv1alpha1.EdgeConnect) *Reconciler {
	return &Reconciler{
		edgeConnect:    edgeConnect,
		apiReader:      apiReader,
		timeProvider:   timeProvider,
		registryClient: registryClient,
	}
}

func (reconciler *Reconciler) Reconcile(ctx context.Context) error {
	updaters := []versionStatusUpdater{
		newUpdater(reconciler.apiReader, reconciler.timeProvider, reconciler.registryClient, reconciler.edgeConnect),
	}

	for _, updater := range updaters {
		log.Info("updating version status", "updater", updater.Name())

		if updater.RequiresReconcile() {
			log.Debug("reconcile required", "updater", updater.Name())
			return updater.Update(ctx)
		}

		log.Info("no reconcile required", "updater", updater.Name())
	}

	return nil
}
