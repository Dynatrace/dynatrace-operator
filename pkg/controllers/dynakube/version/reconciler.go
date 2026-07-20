package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	apiReader client.Reader
}

func NewReconciler(apiReader client.Reader) *Reconciler {
	return &Reconciler{
		apiReader: apiReader,
	}
}

func (r *Reconciler) ReconcileCodeModules(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "version")

	updater := newCodeModulesUpdater(dk, imageClient, versionClient)
	if r.needsUpdate(ctx, updater) {
		return r.updateVersionStatuses(ctx, updater, dk)
	}

	return nil
}

func (r *Reconciler) ReconcileOneAgent(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "version")

	updater := newOneAgentUpdater(dk, r.apiReader, imageClient, versionClient)
	if r.needsUpdate(ctx, updater) {
		return r.updateVersionStatuses(ctx, updater, dk)
	}

	return nil
}

func (r *Reconciler) ReconcileActiveGate(ctx context.Context, dk *dynakube.DynaKube, imageClient image.Client, versionClient version.Client) error {
	ctx, _ = logd.NewFromContext(ctx, "version")

	updater := newActiveGateUpdater(dk, r.apiReader, imageClient, versionClient)
	if r.needsUpdate(ctx, updater) {
		err := r.updateVersionStatuses(ctx, updater, dk)

		return err
	}

	return nil
}

func (r *Reconciler) updateVersionStatuses(ctx context.Context, updater StatusUpdater, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)
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

func (r *Reconciler) needsUpdate(ctx context.Context, updater StatusUpdater) bool {
	log := logd.FromContext(ctx)
	if !updater.IsEnabled() {
		log.Info("skipping version status update for disabled section", "updater", updater.Name())

		return false
	}

	return true
}
