package version

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type oneAgentUpdater struct {
	dynakube       *dynatracev1beta1.DynaKube
	apiReader      client.Reader
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
}

func newOneAgentUpdater(
	dynakube *dynatracev1beta1.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
	registryClient registry.ImageGetter,
) *oneAgentUpdater {
	return &oneAgentUpdater{
		dynakube:       dynakube,
		apiReader:      apiReader,
		dtClient:       dtClient,
		registryClient: registryClient,
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

func (updater oneAgentUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	return updater.dtClient.GetLatestOneAgentImage()
}

func (updater oneAgentUpdater) UseTenantRegistry(ctx context.Context) error {
	var err error
	latestVersion := updater.CustomVersion()
	if latestVersion == "" {
		latestVersion, err = updater.dtClient.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			return err
		}
	}

	downgrade, err := updater.CheckForDowngrade(latestVersion)
	if err != nil || downgrade {
		return err
	}

	defaultImage := updater.dynakube.DefaultOneAgentImage()
	return updateVersionStatusForTenantRegistry(ctx, updater.Target(), updater.registryClient, defaultImage)
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
	case status.PublicRegistryVersionSource:
		previousVersion, err = getTagFromImageID(imageID)
		if err != nil {
			return false, err
		}
	}
	return isDowngrade(updater.Name(), previousVersion, latestVersion)
}
