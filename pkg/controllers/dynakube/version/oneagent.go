package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type oneAgentUpdater struct {
	dynakube  *dynatracev1beta1.DynaKube
	apiReader client.Reader
	dtClient  dtclient.Client
}

func newOneAgentUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
) *oneAgentUpdater {
	return &oneAgentUpdater{
		dynakube:  dynakube,
		apiReader: apiReader,
		dtClient:  dtClient,
	}
}

func (updater oneAgentUpdater) Name() string {
	return "oneagent"
}

func (updater oneAgentUpdater) IsEnabled() bool {
	return updater.dynakube.NeedsOneAgent()
}

func (updater *oneAgentUpdater) Target() *status.VersionStatus {
	return &updater.dynakube.Status.OneAgent.VersionStatus
}

func (updater oneAgentUpdater) CustomImage() string {
	return updater.dynakube.CustomOneAgentImage()
}

func (updater oneAgentUpdater) CustomVersion() string {
	return updater.dynakube.CustomOneAgentVersion()
}

func (updater oneAgentUpdater) IsAutoUpdateEnabled() bool {
	return updater.dynakube.ShouldAutoUpdateOneAgent()
}

func (updater oneAgentUpdater) IsPublicRegistryEnabled() bool {
	return updater.dynakube.FeaturePublicRegistry() && !updater.dynakube.ClassicFullStackMode()
}

func (updater oneAgentUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	return updater.dtClient.GetLatestOneAgentImage(ctx)
}

func (updater oneAgentUpdater) UseTenantRegistry(ctx context.Context) error {
	var err error

	latestVersion := updater.CustomVersion()
	if latestVersion == "" {
		latestVersion, err = updater.dtClient.GetLatestAgentVersion(ctx, dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			log.Info("failed to determine image version")

			return err
		}
	}

	downgrade, err := updater.CheckForDowngrade(latestVersion)
	if err != nil || downgrade {
		return err
	}

	defaultImage := updater.dynakube.DefaultOneAgentImage()

	return updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
}

func (updater *oneAgentUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	imageID := updater.Target().ImageID
	if imageID == "" {
		return false, nil
	}

	var previousVersion string

	var err error

	switch updater.Target().Source {
	case status.TenantRegistryVersionSource:
		previousVersion = updater.Target().Version

		return isDowngrade(updater.Name(), previousVersion, latestVersion)
	case status.PublicRegistryVersionSource:
		previousVersion, err = getTagFromImageID(imageID)
		if err != nil {
			return false, err
		}

		return isDowngrade(updater.Name(), previousVersion, latestVersion)
	}

	return false, nil
}

func (updater oneAgentUpdater) ValidateStatus() error {
	imageVersion := updater.Target().Version
	imageType := updater.Target().Type

	if imageVersion == "" {
		return errors.New("build version of OneAgent image not set")
	}

	if imageType == status.ImmutableImageType {
		if updater.dynakube.ClassicFullStackMode() {
			return errors.New("immutable OneAgent image in combination with classicFullStack mode is not possible")
		}
	}

	log.Info("OneAgent metadata present, image type and image version validated")

	return nil
}
