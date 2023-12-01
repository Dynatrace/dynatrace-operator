package version

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActiveGateImageVersionReconciler struct {
	apiReader      client.Reader
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
	timeProvider   *timeprovider.Provider
}

var _ ImageVersionReconciler = &ActiveGateImageVersionReconciler{}

func initializeVersionStatus(dynakube *dynatracev1beta1.DynaKube, now *metav1.Time) status.VersionStatus {
	result := dynakube.Status.ActiveGate.VersionStatus
	result.LastProbeTimestamp = now
	switch {
	case dynakube.CustomActiveGateImage() != "":
		result.Source = status.CustomImageVersionSource
	case dynakube.FeaturePublicRegistry():
		result.Source = status.PublicRegistryVersionSource
	default:
		result.Source = status.TenantRegistryVersionSource
	}
	return result
}

// Reconcile initializes state used for reconciliation and updates the version status used by the dynakube
func (reconciler *ActiveGateImageVersionReconciler) Reconcile(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	current := initializeVersionStatus(dynakube, reconciler.timeProvider.Now())
	if reconciler.needsUpdate(current, dynakube) {
		updatedStatus, err := reconciler.run(ctx, current, dynakube)
		if err != nil {
			return err
		}
		dynakube.Status.ActiveGate.VersionStatus = updatedStatus
	}
	return nil
}

func (reconciler *ActiveGateImageVersionReconciler) needsUpdate(current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube) bool {
	previous := dynakube.Status.ActiveGate.VersionStatus
	if !dynakube.NeedsActiveGate() {
		log.Info("skipping version status update for disabled section", "updater", "activegate")
		return false
	}

	if current.Source != previous.Source {
		log.Info("source changed, update for version status is needed", "updater", "activegate")
		return true
	}

	if current.CustomImageNeedsReconciliation(logCustomActiveGateImageChangedMessage(current, previous), dynakube.CustomActiveGateImage()) {
		return true
	}

	return isOutdated(logSkipUpdateTimestampValidMessage("activegate"), reconciler.timeProvider, dynakube, dynakube.Status.ActiveGate.LastProbeTimestamp)
}

func logCustomActiveGateImageChangedMessage(current status.VersionStatus, previous status.VersionStatus) func() {
	return func() {
		log.Info("custom image value changed, update for version status is needed", "updater", "activegate", "oldImage", previous.ImageID, "newImage", current.ImageID)
	}
}

func (reconciler *ActiveGateImageVersionReconciler) run(ctx context.Context, current status.VersionStatus, dynakube *dynatracev1beta1.DynaKube) (status.VersionStatus, error) {
	previous := dynakube.Status.ActiveGate.VersionStatus
	if current.Source == status.CustomImageVersionSource {
		log.Info("updating version status according to custom image", "updater", "activegate")

		return updateVersionWithImage(ctx, reconciler.registryClient, current, dynakube.CustomActiveGateImage())
	}

	if !dynakube.FeatureDisableActiveGateUpdates() {
		if InitialUpdateInProgress(previous) {
			log.Info("initial status update in progress with no auto update", "updater", "activegate")
		} else if current.Source == previous.Source {
			log.Info("status updated skipped, due to no auto update", "updater", "activegate")
			return previous, nil
		}
	}

	imageUri, err := getImageUri(dynakube, reconciler.dtClient.GetLatestActiveGateImage, dynakube.DefaultActiveGateImage())
	if err != nil {
		return previous, err
	}
	return updateVersionWithImage(ctx, reconciler.registryClient, current, imageUri)
}
