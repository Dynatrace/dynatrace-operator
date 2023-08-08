package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type activeGateUpdater struct {
	dynakube    *dynatracev1beta1.DynaKube
	dtClient    dtclient.Client
	versionFunc ImageVersionFunc
}

func newActiveGateUpdater(
	dynakube *dynatracev1beta1.DynaKube,
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

func (updater *activeGateUpdater) Target() *status.VersionStatus {
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

func (updater *activeGateUpdater) UseTenantRegistry(ctx context.Context, apiReader client.Reader, registryAuthPath string) error {
	defaultImage := updater.dynakube.DefaultActiveGateImage()
	return updateVersionStatusForTenantRegistry(ctx, apiReader, updater.dynakube, updater.Target(), updater.versionFunc, defaultImage, registryAuthPath)
}
