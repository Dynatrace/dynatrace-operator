package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler interface {
	ReconcileCodeModules(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
	ReconcileOneAgent(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
	ReconcileActiveGate(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
}

type reconciler struct {
	dtClient     dtclient.Client
	timeProvider *timeprovider.Provider

	apiReader client.Reader
}

type ReconcilerBuilder func(apiReader client.Reader, dtClient dtclient.Client, timeProvider *timeprovider.Provider) Reconciler

func NewReconciler(apiReader client.Reader, dtClient dtclient.Client, timeProvider *timeprovider.Provider) Reconciler {
	return &reconciler{
		apiReader:    apiReader,
		timeProvider: timeProvider,
		dtClient:     dtClient,
	}
}

func (r *reconciler) ReconcileCodeModules(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	updater := newCodeModulesUpdater(dynakube, r.dtClient)
	if r.needsUpdate(updater, dynakube) {
		return r.updateVersionStatuses(ctx, updater, dynakube)
	}

	return nil
}

func (r *reconciler) ReconcileOneAgent(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	updater := newOneAgentUpdater(dynakube, r.apiReader, r.dtClient)
	if r.needsUpdate(updater, dynakube) {
		return r.updateVersionStatuses(ctx, updater, dynakube)
	}

	return nil
}

func (r *reconciler) ReconcileActiveGate(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	updaters := []StatusUpdater{
		newActiveGateUpdater(dynakube, r.apiReader, r.dtClient),
	}
	for _, updater := range updaters {
		if r.needsUpdate(updater, dynakube) {
			return r.updateVersionStatuses(ctx, updater, dynakube)
		}
	}

	return nil
}

func (r *reconciler) updateVersionStatuses(ctx context.Context, updater StatusUpdater, dynakube *dynatracev1beta1.DynaKube) error {
	log.Info("updating version status", "updater", updater.Name())

	err := r.run(ctx, updater)
	if err != nil {
		return err
	}

	_, ok := updater.(*oneAgentUpdater)
	if ok {
		healthConfig, err := getOneAgentHealthConfig(dynakube.OneAgentVersion())
		if err != nil {
			log.Error(err, "could not set OneAgent healthcheck")
		} else {
			dynakube.Status.OneAgent.Healthcheck = healthConfig
		}
	}

	return nil
}

func (r *reconciler) needsUpdate(updater StatusUpdater, dynakube *dynatracev1beta1.DynaKube) bool {
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

	if !r.timeProvider.IsOutdated(updater.Target().LastProbeTimestamp, dynakube.FeatureApiRequestThreshold()) {
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
