package version

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type syntheticUpdater struct {
	dynakube *dynatracev1beta1.DynaKube
	dtClient dtclient.Client
	hashFunc ImageHashFunc
}

func newSyntheticUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	dtClient dtclient.Client,
	hashFunc ImageHashFunc,
) *syntheticUpdater {
	return &syntheticUpdater{
		dynakube: dynakube,
		dtClient: dtClient,
		hashFunc: hashFunc,
	}
}

func (updater syntheticUpdater) Name() string {
	return "synthetic"
}

func (updater syntheticUpdater) IsEnabled() bool {
	return updater.dynakube.IsSyntheticMonitoringEnabled()
}

func (updater *syntheticUpdater) Target() *dynatracev1beta1.VersionStatus {
	return &updater.dynakube.Status.Synthetic.VersionStatus
}

func (updater syntheticUpdater) CustomImage() string {
	return updater.dynakube.CustomSyntheticImage()
}

func (updater syntheticUpdater) CustomVersion() string {
	return ""
}

func (updater syntheticUpdater) IsAutoUpdateEnabled() bool {
	return !updater.dynakube.FeatureDisableActiveGateUpdates()
}

func (updater syntheticUpdater) IsPublicRegistryEnabled() bool {
	return false
}

func (updater syntheticUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	return nil, errors.New("unsupported method")
}

func (updater *syntheticUpdater) UseDefaults(ctx context.Context, dockerCfg *dockerconfig.DockerConfig) error {
	defaultImage := updater.dynakube.DefaultSyntheticImage()
	return updateVersionStatus(ctx, updater.Target(), defaultImage, updater.hashFunc, dockerCfg)
}
