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
	activeGateVersionConditionType string = "ActiveGateVersion"
)

type activeGateUpdater struct {
	dk        *dynakube.DynaKube
	apiReader client.Reader
	dtClient  dtclient.Client
}

func newActiveGateUpdater(
	dk *dynakube.DynaKube,
	apiReader client.Reader,
	dtClient dtclient.Client,
) *activeGateUpdater {
	return &activeGateUpdater{
		dk:        dk,
		apiReader: apiReader,
		dtClient:  dtClient,
	}
}

func (updater activeGateUpdater) Name() string {
	return "activegate"
}

func (updater activeGateUpdater) IsEnabled() bool {
	if updater.dk.ActiveGate().IsEnabled() {
		return true
	}

	meta.RemoveStatusCondition(updater.dk.Conditions(), activeGateVersionConditionType)
	updater.dk.Status.ActiveGate.VersionStatus = status.VersionStatus{}

	return false
}

func (updater *activeGateUpdater) Target() *status.VersionStatus {
	return &updater.dk.Status.ActiveGate.VersionStatus
}

func (updater activeGateUpdater) CustomImage() string {
	customImage := updater.dk.ActiveGate().GetCustomImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dk.Conditions(), activeGateVersionConditionType)
	}

	return customImage
}

func (updater activeGateUpdater) CustomVersion() string {
	return "" // can't be set for activeGate
}

func (updater activeGateUpdater) IsAutoUpdateEnabled() bool {
	return !updater.dk.FF().IsActiveGateUpdatesDisabled()
}

func (updater activeGateUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dk.FF().IsPublicRegistry() && !updater.dk.OneAgent().IsClassicFullStackMode()
	if isPublicRegistry {
		setVerifiedCondition(updater.dk.Conditions(), activeGateVersionConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater activeGateUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imageInfo, err := updater.dtClient.GetLatestActiveGateImage(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), activeGateVersionConditionType, err)
	}

	return imageInfo, err
}

func (updater *activeGateUpdater) CheckForDowngrade(_ string) (bool, error) {
	return false, nil
}

func (updater *activeGateUpdater) UseTenantRegistry(ctx context.Context) error {
	latestVersion, err := updater.dtClient.GetLatestActiveGateVersion(ctx, dtclient.OsUnix)
	if err != nil {
		log.Info("failed to determine image version", "error", err)
		k8sconditions.SetDynatraceAPIError(updater.dk.Conditions(), activeGateVersionConditionType, err)

		return err
	}

	defaultImage := updater.dk.ActiveGate().GetDefaultImage(latestVersion)

	err = updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
	if err != nil {
		return err
	}

	setVerifiedCondition(updater.dk.Conditions(), activeGateVersionConditionType)

	return nil
}

func (updater activeGateUpdater) ValidateStatus() error {
	imageVersion := updater.Target().Version
	if imageVersion == "" {
		return errors.New("build version of ActiveGate image is not set")
	}

	return nil
}
