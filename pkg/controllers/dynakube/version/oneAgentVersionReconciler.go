package version

import (
	"context"
	"errors"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OneAgentImageVersionReconciler struct {
	apiReader      client.Reader
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
	timeProvider   *timeprovider.Provider
}

var _ ImageVersionReconciler = &OneAgentImageVersionReconciler{}

// move to versionstatus return versionstatus created provide customimage and oneagentversion as param -> option pattern?
func initializeOneAgentVersionStatus(dynakube *dynatracev1beta1.DynaKube, now *metav1.Time) status.VersionStatus {
	result := dynakube.Status.OneAgent.VersionStatus
	result.LastProbeTimestamp = now
	switch {
	case dynakube.CustomOneAgentImage() != "":
		result.Source = status.CustomImageVersionSource
	case dynakube.FeaturePublicRegistry():
		result.Source = status.PublicRegistryVersionSource
	case dynakube.CustomOneAgentVersion() != "":
		result.Source = status.CustomVersionVersionSource
	default:
		result.Source = status.TenantRegistryVersionSource
	}
	return result
}

// Reconcile initializes state used for reconciliation and updates the version status used by the dynakube
func (reconciler *OneAgentImageVersionReconciler) Reconcile(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	current := initializeOneAgentVersionStatus(dynakube, reconciler.timeProvider.Now())
	if reconciler.needsUpdate(current, dynakube) {
		updatedStatus, err := reconciler.run(ctx, current, dynakube)
		if err != nil {
			return err
		}
		reconciler.SetOneAgentHealthConfig(ctx, dynakube, err)
		if err := ValidateOneAgentStatus(updatedStatus, dynakube); err != nil {
			return err
		}
		dynakube.Status.OneAgent.VersionStatus = updatedStatus
	}
	return nil
}

func ValidateOneAgentStatus(updatedStatus status.VersionStatus, dynakube *dynatracev1beta1.DynaKube) error {
	imageVersion := updatedStatus.Version
	imageType := updatedStatus.Type

	if imageVersion == "" {
		return errors.New("build version of OneAgent image not set")
	}

	if imageType == status.ImmutableImageType {
		if dynakube.ClassicFullStackMode() {
			return errors.New("immutable OneAgent image in combination with classicFullStack mode is not possible")
		} else if dynakube.FeatureDisableReadOnlyOneAgent() {
			return errors.New("immutable OneAgent image in combination with readOnly OneAgent filesystem is not possible")
		}
	}
	log.Info("OneAgent metadata present, image type and image version validated")
	return nil
}

func (reconciler *OneAgentImageVersionReconciler) SetOneAgentHealthConfig(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, err error) {
	healthConfig, err := GetOneAgentHealthConfig(ctx, reconciler.apiReader, reconciler.registryClient, dynakube, dynakube.OneAgentImage())
	if err != nil {
		log.Error(err, "could not set OneAgent healthcheck")
	} else {
		dynakube.Status.OneAgent.Healthcheck = healthConfig
	}
}

// let this exist for every updater
func (reconciler *OneAgentImageVersionReconciler) needsUpdate(current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube) bool {
	previous := dynakube.Status.OneAgent.VersionStatus
	if !dynakube.NeedsOneAgent() {
		log.Info("skipping version status update for disabled section", "updater", "OneAgent")
		return false
	}

	if current.Source != previous.Source {
		log.Info("source changed, update for version status is needed", "updater", "OneAgent")
		return true
	}
	if previous.CustomImageNeedsReconciliation(logCustomOneAgentImageChangedMessage(current, previous), dynakube.CustomOneAgentImage()) {
		return true
	}
	if previous.CustomVersionNeedsReconciliation(logCustomVersionChangedMessage(current, previous), dynakube.CustomOneAgentVersion()) {
		return true
	}

	return isOutdated(logSkipUpdateTimestampValidMessage("OneAgent"), reconciler.timeProvider, dynakube, dynakube.Status.OneAgent.LastProbeTimestamp)
}

func logCustomOneAgentImageChangedMessage(current status.VersionStatus, previous status.VersionStatus) func() {
	return func() {
		log.Info("custom image value changed, update for version status is needed", "updater", "OneAgent", "oldImage", previous.ImageID, "newImage", current.ImageID)
	}
}

func logCustomVersionChangedMessage(current status.VersionStatus, previous status.VersionStatus) func() {
	return func() {
		log.Info("custom version value changed, update for version status is needed", "updater", "OneAgent", "oldVersion", previous.Version, "newVersion", current.Version)
	}
}

func (reconciler *OneAgentImageVersionReconciler) run(ctx context.Context, current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube) (status.VersionStatus, error) {
	previous := dynakube.Status.OneAgent.VersionStatus
	if current.Source == status.CustomImageVersionSource {
		log.Info("updating version status according to custom image", "updater", "OneAgent")

		return updateVersionWithImage(ctx, reconciler.registryClient, current, dynakube.CustomOneAgentImage())
	}

	if !dynakube.ShouldAutoUpdateOneAgent() {
		if InitialUpdateInProgress(previous) {
			log.Info("initial status update in progress with no auto update", "updater", "OneAgent")
		} else if current.Source == previous.Source {
			log.Info("status updated skipped, due to no auto update", "updater", "OneAgent")
			return previous, nil
		}
	}

	if current.Source == status.PublicRegistryVersionSource {
		return reconciler.handlePublicRegistry(ctx, current, dynakube, previous)
	}

	return reconciler.handleTenantRegistry(ctx, current, dynakube, previous)

}

func (reconciler *OneAgentImageVersionReconciler) handleTenantRegistry(ctx context.Context, current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube, previous status.VersionStatus) (status.VersionStatus, error) {
	latestVersion := dynakube.CustomOneAgentVersion()
	var err error
	if latestVersion == "" {
		latestVersion, err = reconciler.dtClient.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			return previous, err
		}
	}

	downgrade, err := CheckForDowngrade(previous, latestVersion)
	if err != nil || downgrade {
		return previous, err
	}
	imageUri, err := getImageUri(dynakube, reconciler.dtClient.GetLatestOneAgentImage, dynakube.DefaultOneAgentImage())
	if err != nil {
		return previous, err
	}
	return updateVersionWithImage(ctx, reconciler.registryClient, current, imageUri)
}

func (reconciler *OneAgentImageVersionReconciler) handlePublicRegistry(ctx context.Context, current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube, previous status.VersionStatus) (status.VersionStatus, error) {
	imageUri, err := getImageUri(dynakube, reconciler.getOneAgentImageAndValidateDowngrade(previous), dynakube.DefaultOneAgentImage())
	if err != nil {
		return previous, err
	}
	newStatus, err := updateVersionWithImage(ctx, reconciler.registryClient, current, imageUri)
	if err != nil {
		return status.VersionStatus{}, err
	}
	return newStatus, nil
}

func (reconciler *OneAgentImageVersionReconciler) getOneAgentImageAndValidateDowngrade(previous status.VersionStatus) func() (*dtclient.LatestImageInfo, error) {
	return func() (*dtclient.LatestImageInfo, error) {
		imageInfo, err := reconciler.dtClient.GetLatestOneAgentImage()
		if err != nil {
			log.Info("could not get public image", "updater", "OneAgent")
			return nil, err
		}
		isDowngrade, err := CheckForDowngrade(previous, imageInfo.Tag)
		if err != nil || isDowngrade {
			return nil, err
		}
		return imageInfo, nil
	}
}

func CheckForDowngrade(previous status.VersionStatus, latestVersion string) (bool, error) {
	imageID := previous.ImageID
	if imageID == "" {
		return false, nil
	}

	var previousVersion string
	var err error
	switch previous.Source {
	case status.TenantRegistryVersionSource:
		previousVersion = previous.Version
	case status.PublicRegistryVersionSource:
		previousVersion, err = getTagFromImageID(imageID)
		if err != nil {
			return false, err
		}
	}
	return isDowngrade("OneAgent", previousVersion, latestVersion)
}
