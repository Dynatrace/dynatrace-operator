package version

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TmpCAPath = "/tmp/dynatrace-operator"
	TmpCAName = "dynatraceCustomCA.crt"
)

type Reconciler struct {
	dynakube     *dynatracev1beta1.DynaKube
	dtClient     dtclient.Client
	hashFunc     ImageHashFunc
	timeProvider *timeprovider.Provider

	fs        afero.Afero
	apiReader client.Reader
}

func NewReconciler(dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, dtClient dtclient.Client, fs afero.Afero, versionProvider ImageHashFunc, timeProvider *timeprovider.Provider) *Reconciler { //nolint:revive
	return &Reconciler{
		dynakube:     dynakube,
		apiReader:    apiReader,
		fs:           fs,
		hashFunc:     versionProvider,
		timeProvider: timeProvider,
		dtClient:     dtClient,
	}
}

// Reconcile updates the version status used by the dynakube
func (reconciler *Reconciler) Reconcile(ctx context.Context) error {
	updaters := []versionStatusUpdater{
		newActiveGateUpdater(reconciler.dynakube, reconciler.dtClient, reconciler.hashFunc),
		newOneAgentUpdater(reconciler.dynakube, reconciler.dtClient, reconciler.hashFunc),
		newCodeModulesUpdater(reconciler.dynakube, reconciler.dtClient),
		newSyntheticUpdater(reconciler.dynakube, reconciler.dtClient, reconciler.hashFunc),
	}

	neededUpdaters := reconciler.needsReconcile(updaters)
	if  len(neededUpdaters) > 0 {
		return reconciler.updateVersionStatuses(ctx, neededUpdaters)
	}
	return nil
}

func (reconciler *Reconciler) updateVersionStatuses(ctx context.Context, updaters []versionStatusUpdater) error {
	dockerConfig, err := reconciler.createDockerConfigWithCustomCAs(ctx)
	if err != nil {
		return err
	}

	defer func(dockerConfig *dockerconfig.DockerConfig, fs afero.Afero) {
		_ = dockerConfig.Cleanup(fs)
	}(dockerConfig, reconciler.fs)

	for _, updater := range updaters {
		log.Info("updating version status", "updater", updater.Name())
		err := reconciler.run(ctx, updater, dockerConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func (reconciler *Reconciler) createDockerConfigWithCustomCAs(ctx context.Context) (*dockerconfig.DockerConfig, error) {
	dockerConfig := dockerconfig.NewDockerConfig(reconciler.apiReader, *reconciler.dynakube)
	err := dockerConfig.StoreRequiredFiles(ctx, reconciler.fs)
	if err != nil {
		log.Info("failed to store required files for docker config")
		return nil, err
	}
	return dockerConfig, nil
}

func (reconciler *Reconciler) needsReconcile(updaters []versionStatusUpdater) []versionStatusUpdater {
	neededUpdaters := []versionStatusUpdater{}
	for _, updater := range updaters {
		if reconciler.needsUpdate(updater) {
			neededUpdaters = append(neededUpdaters, updater)
		}
	}
	return neededUpdaters
}

func (reconciler *Reconciler) needsUpdate(updater versionStatusUpdater) bool {
	if !updater.IsEnabled() {
		log.Info("skipping version status update for disabled section", "updater", updater.Name())
		return false
	}

	if updater.Target().Source != determineSource(updater) {
		log.Info("source changed, update for version status is needed", "updater", updater.Name())
		return true
	}

	if !reconciler.timeProvider.IsOutdated(updater.Target().LastProbeTimestamp, reconciler.dynakube.FeatureApiRequestThreshold()) {
		log.Info("status timestamp still valid, skipping version status updater", "updater", updater.Name())
		return false
	}
	return true
}
