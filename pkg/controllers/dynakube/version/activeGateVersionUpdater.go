package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActiveGateReconciler struct {
	dynakube       *dynatracev1beta1.DynaKube
	dtClient       dtclient.Client
	registryClient registry.ImageGetter
	timeProvider   *timeprovider.Provider
	fs             afero.Afero
	apiReader      client.Reader
}

func NewActiveGateReconciler(dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader, dtClient dtclient.Client, registryClient registry.ImageGetter, fs afero.Afero, timeProvider *timeprovider.Provider) *Reconciler { //nolint:revive
	return &Reconciler{
		dynakube:       dynakube,
		apiReader:      apiReader,
		fs:             fs,
		timeProvider:   timeProvider,
		dtClient:       dtClient,
		registryClient: registryClient,
	}
}

// Reconcile updates the version status used by the dynakube
func (reconciler *ActiveGateReconciler) Reconcile(ctx context.Context) error {

	if reconciler.needsUpdate() {
		return reconciler.run(ctx)
	}
	return nil
}

func (reconciler *ActiveGateReconciler) needsUpdate() bool {
	if !reconciler.dynakube.NeedsActiveGate() {
		log.Info("skipping version status update for disabled section", "updater", "activegate")
		return false
	}

	if reconciler.dynakube.Status.ActiveGate.VersionStatus.Source != determineSourceActiveGate(reconciler.dynakube) {
		log.Info("source changed, update for version status is needed", "updater", "activegate")
		return true
	}

	if hasCustomFieldChangedActiveGate(reconciler.dynakube) {
		return true
	}

	if !reconciler.timeProvider.IsOutdated(reconciler.dynakube.Status.ActiveGate.VersionStatus.LastProbeTimestamp, reconciler.dynakube.FeatureApiRequestThreshold()) {
		log.Info("status timestamp still valid, skipping version status updater", "updater", "activegate")
		return false
	}
	return true
}

func hasCustomFieldChangedActiveGate(dynakube *dynatracev1beta1.DynaKube) bool {
	updaterTarget := dynakube.Status.ActiveGate.VersionStatus
	if updaterTarget.Source == status.CustomImageVersionSource {
		oldImage := updaterTarget.ImageID
		newImage := dynakube.CustomActiveGateImage()
		// The old image is can be the same as the new image (if only digest was given, or a tag was given but couldn't get the digest)
		// or the old image is the same as the new image but with the digest added to the end of it (if a tag was provide, and we could append the digest to the end)
		// or the 2 images are different
		if !strings.HasPrefix(oldImage, newImage) {
			log.Info("custom image value changed, update for version status is needed", "updater", "activegate", "oldImage", oldImage, "newImage", newImage)
			return true
		}
	}
	return false
}

func (reconciler *ActiveGateReconciler) run(ctx context.Context) error {
	updaterTarget := reconciler.dynakube.Status.ActiveGate.VersionStatus
	currentSource := determineSourceActiveGate(reconciler.dynakube)
	var err error
	defer func() {
		if err == nil {
			updaterTarget.LastProbeTimestamp = reconciler.timeProvider.Now()
			updaterTarget.Source = currentSource
		}
	}()

	customImage := reconciler.dynakube.CustomActiveGateImage()
	if customImage != "" {
		log.Info("updating version status according to custom image", "updater", "activegate")
		err = setImageIDWithDigest(ctx, &reconciler.dynakube.Status.ActiveGate.VersionStatus, reconciler.registryClient, customImage)
		if err != nil {
			return err
		}
		return activeGateValidateStatus(reconciler.dynakube)
	}

	if !reconciler.dynakube.FeatureDisableActiveGateUpdates() {
		previousSource := reconciler.dynakube.Status.ActiveGate.VersionStatus.Source
		emptyVersionStatus := status.VersionStatus{}
		if &reconciler.dynakube.Status.ActiveGate.VersionStatus == nil || reconciler.dynakube.Status.ActiveGate.VersionStatus == emptyVersionStatus {
			log.Info("initial status update in progress with no auto update", "updater", "activegate")
		} else if previousSource == currentSource {
			log.Info("status updated skipped, due to no auto update", "updater", "activegate")
			return nil
		}
	}

	if reconciler.dynakube.FeaturePublicRegistry() {
		err = reconciler.processPublicRegistry(ctx)
		if err != nil {
			return err
		}
		return activeGateValidateStatus(reconciler.dynakube)
	}

	log.Info("updating version status according to the tenant registry", "updater", "activegate")
	defaultImage := reconciler.dynakube.DefaultActiveGateImage()
	err = updateVersionStatusForTenantRegistry(ctx, updaterTarget, reconciler.registryClient, defaultImage)
	if err != nil {
		return err
	}

	return activeGateValidateStatus(reconciler.dynakube)
}

func activeGateValidateStatus(dynakube *dynatracev1beta1.DynaKube) error {
	imageVersion := dynakube.Status.ActiveGate.VersionStatus.Version
	if imageVersion == "" {
		return errors.New("build version of ActiveGate image is not set")
	}
	return nil
}

func determineSourceActiveGate(dynakube *dynatracev1beta1.DynaKube) status.VersionSource {
	if dynakube.CustomActiveGateImage() != "" {
		return status.CustomImageVersionSource
	}
	if dynakube.FeaturePublicRegistry() {
		return status.PublicRegistryVersionSource
	}
	return status.TenantRegistryVersionSource
}

func (reconciler *ActiveGateReconciler) processPublicRegistry(ctx context.Context) error {
	log.Info("updating version status according to public registry", "updater", "activegate")
	var publicImage *dtclient.LatestImageInfo
	publicImage, err := reconciler.dtClient.GetLatestActiveGateImage()
	if err != nil {
		log.Info("could not get public image", "updater", "activegate")
		return err
	}

	err = setImageIDWithDigest(ctx, &reconciler.dynakube.Status.ActiveGate.VersionStatus, reconciler.registryClient, publicImage.String())
	if err != nil {
		log.Info("could not update version status according to the public registry", "updater", "activegate")
		return err
	}
	return nil
}
