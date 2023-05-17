package version

import (
	"context"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type activeGateUpdater struct {
	dynakube    *dynatracev1.DynaKube
	dtClient    dtclient.Client
	versionFunc ImageVersionFunc
}

func newActiveGateUpdater(
	dynakube *dynatracev1.DynaKube,
	dtClient dtclient.Client,
	versionFunc ImageVersionFunc,
) *activeGateUpdater {
	return &activeGateUpdater{
		dynakube:    dynakube,
		dtClient:    dtClient,
		versionFunc: versionFunc,
	}
}

func (updater activeGateUpdater) Name() string {
	return "activegate"
}

func (updater activeGateUpdater) IsEnabled() bool {
	return updater.dynakube.NeedsActiveGate()
}

func (updater *activeGateUpdater) Target() *dynatracev1.VersionStatus {
	return &updater.dynakube.Status.ActiveGate.VersionStatus
}

func (updater activeGateUpdater) CustomImage() string {
	return updater.dynakube.CustomActiveGateImage()
}

func (updater activeGateUpdater) CustomVersion() string {
	return "" // can't be set for activeGate
}

func (updater activeGateUpdater) IsAutoUpdateEnabled() bool {
	return !updater.dynakube.FeatureDisableActiveGateUpdates()
}

func (updater activeGateUpdater) IsPublicRegistryEnabled() bool {
	return updater.dynakube.FeaturePublicRegistry()
}

func (updater activeGateUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	return updater.dtClient.GetLatestActiveGateImage()
}

func (updater *activeGateUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	return false, nil
}

func (updater *activeGateUpdater) UseTenantRegistry(ctx context.Context, dockerCfg *dockerconfig.DockerConfig) error {
	defaultImage := updater.dynakube.DefaultActiveGateImage()
	return updateVersionStatusForTenantRegistry(ctx, updater.Target(), defaultImage, updater.versionFunc, dockerCfg, updater.dynakube)
}
