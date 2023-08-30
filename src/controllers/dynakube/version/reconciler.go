package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
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
	versionFunc  ImageVersionFunc
	timeProvider *timeprovider.Provider

	fs        afero.Afero
	apiReader client.Reader
}

func NewReconciler(dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, dtClient dtclient.Client, fs afero.Afero, digestProvider ImageVersionFunc, timeProvider *timeprovider.Provider) *Reconciler { //nolint:revive
	return &Reconciler{
		dynakube:     dynakube,
		apiReader:    apiReader,
		fs:           fs,
		versionFunc:  digestProvider,
		timeProvider: timeProvider,
		dtClient:     dtClient,
	}
}

// Reconcile updates the version status used by the dynakube
func (reconciler *Reconciler) Reconcile(ctx context.Context) error {
	updaters := []versionStatusUpdater{
		newActiveGateUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.versionFunc),
		newOneAgentUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.versionFunc),
		newCodeModulesUpdater(reconciler.dynakube, reconciler.dtClient),
		newSyntheticUpdater(reconciler.dynakube, reconciler.apiReader, reconciler.dtClient, reconciler.versionFunc),
	}

	neededUpdaters := reconciler.needsReconcile(updaters)
	if len(neededUpdaters) > 0 {
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
		err := reconciler.run(ctx, updater)
		if err != nil {
			return err
		}
	}

	healthConfig, err := GetOneAgentHealthConfig(ctx, reconciler.apiReader, registry.NewClient(), reconciler.dynakube, reconciler.dynakube.OneAgentImage())
	if err != nil {
		log.Error(err, "could not set OneAgent healthcheck")
	} else {
		reconciler.dynakube.Status.OneAgent.Healthcheck = healthConfig
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

	if hasCustomFieldChanged(updater) {
		return true
	}

	if !reconciler.timeProvider.IsOutdated(updater.Target().LastProbeTimestamp, reconciler.dynakube.FeatureApiRequestThreshold()) {
		log.Info("status timestamp still valid, skipping version status updater", "updater", updater.Name())
		return false
	}
	return true
}

func hasCustomFieldChanged(updater versionStatusUpdater) bool {
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
