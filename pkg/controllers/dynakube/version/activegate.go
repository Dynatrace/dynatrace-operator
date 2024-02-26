package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type activeGateUpdater struct {
	dynakube  *dynatracev1beta1.DynaKube
	apiReader client.Reader
	dtClient  dtclient.Client
}

func newActiveGateUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
) *activeGateUpdater {
	return &activeGateUpdater{
		dynakube:  dynakube,
		apiReader: apiReader,
		dtClient:  dtClient,
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

func (updater activeGateUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	return updater.dtClient.GetLatestActiveGateImage(ctx)
}

func (updater *activeGateUpdater) CheckForDowngrade(_ string) (bool, error) {
	return false, nil
}

func (updater *activeGateUpdater) UseTenantRegistry(ctx context.Context) error {
	latestVersion, err := updater.dtClient.GetLatestActiveGateVersion(ctx, dtclient.OsUnix)
	if err != nil {
		log.Info("failed to determine image version", "error", err)

		return err
	}

	defaultImage := updater.dynakube.DefaultActiveGateImage()

	return updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
}

func (updater activeGateUpdater) ValidateStatus() error {
	imageVersion := updater.Target().Version
	if imageVersion == "" {
		return errors.New("build version of ActiveGate image is not set")
	}

	return nil
}
