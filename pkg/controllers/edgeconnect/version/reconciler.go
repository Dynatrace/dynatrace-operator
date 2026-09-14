// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type versionStatusUpdater interface {
	Name() string
	RequiresReconcile() bool
	Update(ctx context.Context) error
}

// ImageClientProvider builds the fleet management image client on demand. Building it requires an
// OAuth token exchange, so it is only invoked once an image actually has to be resolved.
type ImageClientProvider func(ctx context.Context) (dtimage.Client, error)

// RegistryClientProvider builds the OCI registry client on demand. Building it reads the pull
// secret, which the fleet management path does not need, so it is only invoked once the fallback
// actually has to be used.
type RegistryClientProvider func(ctx context.Context) (registry.ImageGetter, error)

type Reconciler struct {
	edgeConnect  *edgeconnect.EdgeConnect
	timeProvider *timeprovider.Provider

	apiReader              client.Reader
	imageClientProvider    ImageClientProvider
	registryClientProvider RegistryClientProvider
}

func NewReconciler(apiReader client.Reader, imageClientProvider ImageClientProvider, registryClientProvider RegistryClientProvider, timeProvider *timeprovider.Provider, ec *edgeconnect.EdgeConnect) *Reconciler {
	return &Reconciler{
		edgeConnect:            ec,
		apiReader:              apiReader,
		timeProvider:           timeProvider,
		imageClientProvider:    imageClientProvider,
		registryClientProvider: registryClientProvider,
	}
}

func (reconciler *Reconciler) Reconcile(ctx context.Context) error {
	ctx, log := logd.NewFromContext(ctx, "version")

	updaters := []versionStatusUpdater{
		newUpdater(reconciler.apiReader, reconciler.timeProvider, reconciler.imageClientProvider, reconciler.registryClientProvider, reconciler.edgeConnect),
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
