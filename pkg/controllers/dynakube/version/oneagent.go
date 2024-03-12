package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	oaConditionType = "OneAgentVersion"
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
	if updater.dynakube.NeedsOneAgent() {
		return true
	}

	_ = meta.RemoveStatusCondition(updater.dynakube.Conditions(), oaConditionType)

	return updater.dynakube.NeedsOneAgent()
}

func (updater *oneAgentUpdater) Target() *status.VersionStatus {
	return &updater.dynakube.Status.OneAgent.VersionStatus
}

func (updater oneAgentUpdater) CustomImage() string {
	customImage := updater.dynakube.CustomOneAgentImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dynakube.Conditions(), oaConditionType)
	}

	return customImage
}

func (updater oneAgentUpdater) CustomVersion() string {
	return updater.dynakube.CustomOneAgentVersion()
}

func (updater oneAgentUpdater) IsAutoUpdateEnabled() bool {
	return updater.dynakube.ShouldAutoUpdateOneAgent()
}

func (updater oneAgentUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dynakube.FeaturePublicRegistry() && !updater.dynakube.ClassicFullStackMode()
	if isPublicRegistry {
		setVerifiedCondition(updater.dynakube.Conditions(), oaConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater oneAgentUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imageInfo, err := updater.dtClient.GetLatestOneAgentImage(ctx)
	if err != nil {
		conditions.SetDynatraceApiErrorCondition(updater.dynakube.Conditions(), oaConditionType, err)
	}

	return imageInfo, err
}

func (updater oneAgentUpdater) UseTenantRegistry(ctx context.Context) error {
	var err error

	// Not using setVerificationSkippedReasonCondition here because technically we do some verification.
	latestVersion := updater.CustomVersion()

	if latestVersion == "" {
		latestVersion, err = updater.dtClient.GetLatestAgentVersion(ctx, dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			log.Info("failed to determine image version")
			conditions.SetDynatraceApiErrorCondition(updater.dynakube.Conditions(), oaConditionType, err)

			return err
		}
	}

	downgrade, err := updater.CheckForDowngrade(latestVersion)
	if err != nil || downgrade {
		return err
	}

	defaultImage := updater.dynakube.DefaultOneAgentImage(latestVersion)

	err = updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
	if err != nil {
		return err
	}

	setVerifiedCondition(updater.dynakube.Conditions(), oaConditionType)

	return nil
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
			setVerificationFailedReasonCondition(updater.dynakube.Conditions(), oaConditionType, err)

			return false, err
		}
	}

	downgrade, err := isDowngrade(updater.Name(), previousVersion, latestVersion)
	if downgrade {
		setDowngradeCondition(updater.dynakube.Conditions(), oaConditionType, previousVersion, latestVersion)
	}

	if err != nil {
		setVerificationFailedReasonCondition(updater.dynakube.Conditions(), oaConditionType, err)
	}

	return downgrade, err
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
