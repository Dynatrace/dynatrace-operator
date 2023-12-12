package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	dynakube       *dynatracev1beta1.DynaKube
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
	timeProvider   *timeprovider.Provider

	fs        afero.Afero
	apiReader client.Reader
}

func NewReconciler(dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, dtClient dtclient.Client, registryClient registry.ImageGetter, fs afero.Afero, timeProvider *timeprovider.Provider) *Reconciler { //nolint:revive
	return &Reconciler{
		dynakube:       dynakube,
		apiReader:      apiReader,
		fs:             fs,
		timeProvider:   timeProvider,
		dtClient:       dtClient,
		registryClient: registryClient,
	}
}

func (reconciler *Reconciler) ReconcileCodeModules(ctx context.Context) error {
	updater := newCodeModulesUpdater(reconciler.dynakube, reconciler.dtClient)
	if reconciler.needsUpdate(updater) {
		return reconciler.updateVersionStatuses(ctx, updater)
	}
	return nil
}

func (reconciler *Reconciler) ReconcileOneAgent(ctx context.Context) error {
	updater := newOneAgentUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.registryClient)
	if reconciler.needsUpdate(updater) {
		return reconciler.updateVersionStatuses(ctx, updater)
	}
	return nil
}

func (reconciler *Reconciler) ReconcileActiveGate(ctx context.Context) error {
	updaters := []StatusUpdater{
		newActiveGateUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.registryClient),
		newSyntheticUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.registryClient),
	}
	for _, updater := range updaters {
		if reconciler.needsUpdate(updater) {
			return reconciler.updateVersionStatuses(ctx, updater)
		}
	}
	return nil
}

func (reconciler *Reconciler) updateVersionStatuses(ctx context.Context, updater StatusUpdater) error {
	log.Info("updating version status", "updater", updater.Name())
	err := reconciler.run(ctx, updater)
	if err != nil {
		return err
	}

	_, ok := updater.(*oneAgentUpdater)
	if ok {
		healthConfig, err := GetOneAgentHealthConfig(ctx, reconciler.apiReader, reconciler.registryClient, reconciler.dynakube, reconciler.dynakube.OneAgentImage())
		if err != nil {
			log.Error(err, "could not set OneAgent healthcheck")
		} else {
			reconciler.dynakube.Status.OneAgent.Healthcheck = healthConfig
		}
	}
	return nil
}

func (reconciler *Reconciler) needsUpdate(updater StatusUpdater) bool {
	if !updater.IsEnabled() {
		log.Info("skipping version status update for disabled section", "updater", updater.Name())
		return false
	}

	if updater.Target().Source != determineSource(updater) {
		log.Info("source changed, update for version status is needed", "updater", updater.Name())
		return true
	}

	if hasCustomFieldChanged(updater) {
		return true
	}

	if !reconciler.timeProvider.IsOutdated(updater.Target().LastProbeTimestamp, reconciler.dynakube.FeatureApiRequestThreshold()) {
		log.Info("status timestamp still valid, skipping version status updater", "updater", updater.Name())
		return false
	}
	return true
}

func hasCustomFieldChanged(updater StatusUpdater) bool {
	if updater.Target().Source == status.CustomImageVersionSource {
		oldImage := updater.Target().ImageID
		newImage := updater.CustomImage()
		// The old image is can be the same as the new image (if only digest was given, or a tag was given but couldn't get the digest)
		// or the old image is the same as the new image but with the digest added to the end of it (if a tag was provide, and we could append the digest to the end)
		// or the 2 images are different
		if !strings.HasPrefix(oldImage, newImage) {
			log.Info("custom image value changed, update for version status is needed", "updater", updater.Name(), "oldImage", oldImage, "newImage", newImage)
			return true
		}
	} else if updater.Target().Source == status.CustomVersionVersionSource {
		oldVersion := updater.Target().Version
		newVersion := updater.CustomVersion()
		if oldVersion != newVersion {
			log.Info("custom version value changed, update for version status is needed", "updater", updater.Name(), "oldVersion", oldVersion, "newVersion", newVersion)
			return true
		}
	}
	return false
}
