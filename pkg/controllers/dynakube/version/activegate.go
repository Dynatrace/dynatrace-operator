package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	activeGateVersionConditionType string = "ActiveGateVersion"
)

type activeGateUpdater struct {
	dynakube  *dynatracev1beta2.DynaKube
	apiReader client.Reader
	dtClient  dtclient.Client
}

func newActiveGateUpdater(
	dynakube *dynatracev1beta2.DynaKube,
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
	if updater.dynakube.NeedsActiveGate() {
		return true
	}

	meta.RemoveStatusCondition(updater.dynakube.Conditions(), activeGateVersionConditionType)
	updater.dynakube.Status.ActiveGate.VersionStatus = status.VersionStatus{}

	return false
}

func (updater *activeGateUpdater) Target() *status.VersionStatus {
	return &updater.dynakube.Status.ActiveGate.VersionStatus
}

func (updater activeGateUpdater) CustomImage() string {
	customImage := updater.dynakube.CustomActiveGateImage()
	if customImage != "" {
		setVerificationSkippedReasonCondition(updater.dynakube.Conditions(), activeGateVersionConditionType)
	}

	return customImage
}

func (updater activeGateUpdater) CustomVersion() string {
	return "" // can't be set for activeGate
}

func (updater activeGateUpdater) IsAutoUpdateEnabled() bool {
	return !updater.dynakube.FeatureDisableActiveGateUpdates()
}

func (updater activeGateUpdater) IsPublicRegistryEnabled() bool {
	isPublicRegistry := updater.dynakube.FeaturePublicRegistry() && !updater.dynakube.ClassicFullStackMode()
	if isPublicRegistry {
		setVerifiedCondition(updater.dynakube.Conditions(), activeGateVersionConditionType) // Bit hacky, as things can still go wrong, but if so we will just overwrite this is LatestImageInfo.
	}

	return isPublicRegistry
}

func (updater activeGateUpdater) LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error) {
	imageInfo, err := updater.dtClient.GetLatestActiveGateImage(ctx)
	if err != nil {
		conditions.SetDynatraceApiError(updater.dynakube.Conditions(), activeGateVersionConditionType, err)
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
		conditions.SetDynatraceApiError(updater.dynakube.Conditions(), activeGateVersionConditionType, err)

		return err
	}

	defaultImage := updater.dynakube.DefaultActiveGateImage(latestVersion)

	err = updateVersionStatusForTenantRegistry(updater.Target(), defaultImage, latestVersion)
	if err != nil {
		return err
	}

	setVerifiedCondition(updater.dynakube.Conditions(), activeGateVersionConditionType)

	return nil
}

func (updater activeGateUpdater) ValidateStatus() error {
	imageVersion := updater.Target().Version
	if imageVersion == "" {
		return errors.New("build version of ActiveGate image is not set")
	}

	return nil
}
