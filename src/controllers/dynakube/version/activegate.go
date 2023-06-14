package version

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type activeGateUpdater struct {
	dynakube         *dynatracev1beta1.DynaKube
	dtClient         dtclient.Client
	versionProxyFunc ImageVersionProxyFunc
}

func newActiveGateUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	dtClient dtclient.Client,
	versionProxyFunc ImageVersionProxyFunc,
) *activeGateUpdater {
	return &activeGateUpdater{
		dynakube:         dynakube,
		dtClient:         dtClient,
		versionProxyFunc: versionProxyFunc,
	}
}

func (updater activeGateUpdater) Name() string {
	return "activegate"
}

func (updater activeGateUpdater) IsEnabled() bool {
	return updater.dynakube.NeedsActiveGate()
}

func (updater *activeGateUpdater) Target() *dynatracev1beta1.VersionStatus {
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
	return updateVersionStatusForTenantRegistryViaProxy(ctx, updater.Target(), defaultImage, updater.versionProxyFunc, dockerCfg, updater.dynakube)
}
