package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler interface {
	ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube) error
	ReconcileOneAgent(ctx context.Context, dk *dynakube.DynaKube) error
	ReconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube) error
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

func (r *reconciler) ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube) error {
	updater := newCodeModulesUpdater(dk, r.dtClient)
	if r.needsUpdate(updater) {
		return r.updateVersionStatuses(ctx, updater, dk)
	}

	return nil
}

func (r *reconciler) ReconcileOneAgent(ctx context.Context, dk *dynakube.DynaKube) error {
	updater := newOneAgentUpdater(dk, r.apiReader, r.dtClient)
	if r.needsUpdate(updater) {
		return r.updateVersionStatuses(ctx, updater, dk)
	}

	return nil
}

func (r *reconciler) ReconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube) error {
	updater := newActiveGateUpdater(dk, r.apiReader, r.dtClient)
	if r.needsUpdate(updater) {
		err := r.updateVersionStatuses(ctx, updater, dk)

		return err
	}

	return nil
}

func (r *reconciler) updateVersionStatuses(ctx context.Context, updater StatusUpdater, dk *dynakube.DynaKube) error {
	log.Info("updating version status", "updater", updater.Name())

	err := r.run(ctx, updater)
	if err != nil {
		if updater.Target().ImageID == "" && updater.Target().Version == "" {
			log.Info("unable to set version info, no previous version to fallback to", "component", updater.Name())

			return err
		}

		log.Error(err, "unable to refresh version info, moving on with version from previous run", "component", updater.Name())
	}

	_, ok := updater.(*oneAgentUpdater)
	if ok {
		healthConfig, err := getOneAgentHealthConfig(dk.OneAgent().GetVersion())
		if err != nil {
			log.Error(err, "could not set OneAgent healthcheck")
		} else {
			dk.Status.OneAgent.Healthcheck = healthConfig
		}
	}

	return nil
}

func (r *reconciler) needsUpdate(updater StatusUpdater) bool {
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
