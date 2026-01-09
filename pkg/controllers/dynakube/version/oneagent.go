package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	oaConditionType = "OneAgentVersion"
)

type oneAgentUpdater struct {
	dk        *dynakube.DynaKube
	apiReader client.Reader
	dtClient  dtclient.Client
}

func newOneAgentUpdater(
	dk *dynakube.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
) *oneAgentUpdater {
	return &oneAgentUpdater{
		dk:        dk,
		apiReader: apiReader,
		dtClient:  dtClient,
	}
}

func (updater oneAgentUpdater) Name() string {
	return "oneagent"
}

func (updater oneAgentUpdater) IsEnabled() bool {
	if updater.dk.OneAgent().IsDaemonsetRequired() {
		return true
	}

	updater.dk.Status.OneAgent.VersionStatus = status.VersionStatus{}
	updater.dk.Status.OneAgent.Healthcheck = nil
	_ = meta.RemoveStatusCondition(updater.dk.Conditions(), oaConditionType)

	return updater.dk.OneAgent().IsDaemonsetRequired()
}

func (updater *oneAgentUpdater) Target() *status.VersionStatus {
	return &updater.dk.Status.OneAgent.VersionStatus
}

func (updater oneAgentUpdater) CustomImage() string {
	customImage := updater.dk.OneAgent().GetCustomImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dk.Conditions(), oaConditionType)
	}

	return customImage
}

func (updater oneAgentUpdater) CustomVersion() string {
	return updater.dk.OneAgent().GetCustomVersion()
}

func (updater oneAgentUpdater) IsAutoUpdateEnabled() bool {
	return updater.dk.OneAgent().IsAutoUpdateEnabled()
}

func (updater oneAgentUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dk.FF().IsPublicRegistry() && !updater.dk.OneAgent().IsClassicFullStackMode()
	if isPublicRegistry {
		setVerifiedCondition(updater.dk.Conditions(), oaConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater oneAgentUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imageInfo, err := updater.dtClient.GetLatestOneAgentImage(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), oaConditionType, err)
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
			k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), oaConditionType, err)

			return err
		}
	}

	downgrade, err := updater.CheckForDowngrade(latestVersion)
	if err != nil || downgrade {
		return err
	}

	defaultImage := updater.dk.OneAgent().GetDefaultImage(latestVersion)

	err = updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
	if err != nil {
		return err
	}

	setVerifiedCondition(updater.dk.Conditions(), oaConditionType)

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
			setVerificationFailedReasonCondition(updater.dk.Conditions(), oaConditionType, err)

			return false, err
		}
	}

	downgrade, err := isDowngrade(updater.Name(), previousVersion, latestVersion)
	if downgrade {
		setDowngradeCondition(updater.dk.Conditions(), oaConditionType, previousVersion, latestVersion)
	}

	if err != nil {
		setVerificationFailedReasonCondition(updater.dk.Conditions(), oaConditionType, err)
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
		if updater.dk.OneAgent().IsClassicFullStackMode() {
			return errors.New("immutable OneAgent image in combination with classicFullStack mode is not possible")
		}
	}

	log.Info("OneAgent metadata present, image type and image version validated")

	return nil
}
