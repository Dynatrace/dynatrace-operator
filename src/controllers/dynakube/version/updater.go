package version

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type versionStatusUpdater interface {
	Name() string
	IsEnabled() bool
	Target() *dynatracev1beta1.VersionStatus

	CustomImage() string
	CustomVersion() string
	IsAutoUpdateEnabled() bool
	LatestImageInfo() (*dtclient.LatestImageInfo, error)

	UseDefaults(context.Context, *dockerconfig.DockerConfig) error
}

func (reconciler *Reconciler) run(ctx context.Context, updater versionStatusUpdater, dockerCfg *dockerconfig.DockerConfig) error {
	currentSource := reconciler.determineSource(updater)
	var err error
	defer func() {
		if err == nil {
			updater.Target().LastProbeTimestamp = reconciler.timeProvider.Now()
			updater.Target().Source = currentSource
		}
	}()

	customImage := dtclient.ImageInfoFromUri(updater.CustomImage())
	if customImage != nil {
		err = updateVersionStatus(ctx, updater.Target(), customImage, reconciler.hashFunc, dockerCfg)
		return err
	}

	if !updater.IsAutoUpdateEnabled() {
		emptyVersionStatus := dynatracev1beta1.VersionStatus{}
		previousSource := updater.Target().Source
		if updater.Target() == nil || *updater.Target() == emptyVersionStatus {
			log.Info("initial status update in progress with no auto update", "updater", updater.Name())
		} else if previousSource == currentSource {
			log.Info("status updated skipped, due to no auto update", "updater", updater.Name())
			return nil
		}
	}

	if reconciler.dynakube.FeaturePublicRegistry() {
		var publicImage *dtclient.LatestImageInfo
		publicImage, err = updater.LatestImageInfo()
		if err != nil {
			log.Info("could not get public image", "updater", updater.Name())
			return err
		}
		err = updateVersionStatus(ctx, updater.Target(), publicImage, reconciler.hashFunc, dockerCfg)
		if err != nil {
			log.Info("could not update version status according to the public registry", "updater", updater.Name())
			return err
		}
		updater.Target().Version = updater.Target().ImageTag
		return nil
	}

	err = updater.UseDefaults(ctx, dockerCfg)
	return err
}

func (reconciler *Reconciler) determineSource(updater versionStatusUpdater) dynatracev1beta1.VersionSource {
	if updater.CustomImage() != "" {
		return dynatracev1beta1.CustomImageVersionSource
	}
	if reconciler.dynakube.FeaturePublicRegistry() {
		return dynatracev1beta1.PublicRegistryVersionSource
	}
	if updater.CustomVersion() != "" {
		return dynatracev1beta1.CustomVersionVersionSource
	}
	return dynatracev1beta1.DefaultVersionSource
}

func updateVersionStatus(
	ctx context.Context,
	target *dynatracev1beta1.VersionStatus,
	image *dtclient.LatestImageInfo,
	hashFunc ImageHashFunc,
	dockerCfg *dockerconfig.DockerConfig,
) error {
	imageUri := image.Uri()

	hash, err := hashFunc(ctx, imageUri, dockerCfg)
	if err != nil {
		return errors.WithMessage(err, "failed to get image hash")
	}

	log.Info("checked image version info",
		"image", image,
		"oldTag", target.ImageTag, "newTag", image.Tag,
		"oldHash", target.ImageHash, "newHash", hash)

	target.ImageTag = image.Tag
	target.ImageHash = hash
	target.ImageRepository = image.Source
	// Version will be set elsewhere, as it differs between modes
	// unset is necessary so we have a consistent status
	target.Version = ""
	return nil
}
