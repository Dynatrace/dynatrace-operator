package version

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
)

type versionStatusUpdater interface {
	Name() string
	IsEnabled() bool
	Target() *dynatracev1beta1.VersionStatus

	CustomImage() string
	CustomVersion() string
	IsClassicFullStackEnabled() bool
	IsAutoUpdateEnabled() bool
	IsPublicRegistryEnabled() bool
	LatestImageInfo() (*dtclient.LatestImageInfo, error)

	UseDefaults(context.Context, *dockerconfig.DockerConfig) error
}

func (reconciler *Reconciler) run(ctx context.Context, updater versionStatusUpdater, dockerCfg *dockerconfig.DockerConfig) error {
	currentSource := determineSource(updater)
	var err error
	defer func() {
		if err == nil {
			updater.Target().LastProbeTimestamp = reconciler.timeProvider.Now()
			updater.Target().Source = currentSource
		}
	}()

	customImage := updater.CustomImage()
	if customImage != "" {
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

	if updater.IsPublicRegistryEnabled() && !updater.IsClassicFullStackEnabled() {
		var publicImage *dtclient.LatestImageInfo
		publicImage, err = updater.LatestImageInfo()
		if err != nil {
			log.Info("could not get public image", "updater", updater.Name())
			return err
		}
		err = updateVersionStatus(ctx, updater.Target(), publicImage.String(), reconciler.hashFunc, dockerCfg)
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

func determineSource(updater versionStatusUpdater) dynatracev1beta1.VersionSource {
	if updater.CustomImage() != "" {
		return dynatracev1beta1.CustomImageVersionSource
	}
	if updater.IsPublicRegistryEnabled() && !updater.IsClassicFullStackEnabled() {
		return dynatracev1beta1.PublicRegistryVersionSource
	}
	if updater.CustomVersion() != "" {
		return dynatracev1beta1.CustomVersionVersionSource
	}
	return dynatracev1beta1.TenantRegistryVersionSource
}

func updateVersionStatus(
	ctx context.Context,
	target *dynatracev1beta1.VersionStatus,
	imageUri string,
	hashFunc ImageHashFunc,
	dockerCfg *dockerconfig.DockerConfig,
) error {
	ref, err := reference.Parse(imageUri)
	if err != nil {
		return errors.WithMessage(err, "failed to parse image uri")
	}

	var repo, hash, tag string
	if canonRef, ok := ref.(reference.Canonical); ok {
		repo = canonRef.Name()
		hash = canonRef.Digest().String()
		tag = hash
	} else if taggedRef, ok := ref.(reference.NamedTagged); ok {
		repo = taggedRef.Name()
		tag = taggedRef.Tag()
		hash, err = hashFunc(ctx, imageUri, dockerCfg)
		if err != nil {
			return errors.WithMessage(err, "failed to get image hash")
		}
	}

	log.Info("checked image version info",
		"image", imageUri,
		"oldRepo", target.ImageRepository, "newRepo", repo,
		"oldTag", target.ImageTag, "newTag", tag,
		"oldHash", target.ImageHash, "newHash", hash)

	target.ImageTag = tag
	target.ImageHash = hash
	target.ImageRepository = repo
	// Version will be set elsewhere, as it differs between modes
	// unset is necessary so we have a consistent status
	target.Version = ""
	return nil
}
