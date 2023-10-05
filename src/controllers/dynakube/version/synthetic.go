package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type syntheticUpdater struct {
	dynakube       *dynatracev1beta1.DynaKube
	apiReader      client.Reader
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
}

func newSyntheticUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
	registryClient registry.ImageGetter,
) *syntheticUpdater {
	return &syntheticUpdater{
		dynakube:       dynakube,
		apiReader:      apiReader,
		dtClient:       dtClient,
		registryClient: registryClient,
	}
}

func (updater syntheticUpdater) Name() string {
	return "synthetic"
}

func (updater syntheticUpdater) IsEnabled() bool {
	return updater.dynakube.IsSyntheticMonitoringEnabled()
}

func (updater *syntheticUpdater) Target() *status.VersionStatus {
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

func (updater syntheticUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	return false, nil
}

func (updater *syntheticUpdater) UseTenantRegistry(ctx context.Context) error {
	defaultImage := updater.dynakube.DefaultSyntheticImage()
	return updateVersionStatusForTenantRegistry(ctx, updater.Target(), updater.registryClient, defaultImage)
}
