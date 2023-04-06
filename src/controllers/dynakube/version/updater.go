package version

import (
	"context"
	"fmt"

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
	IsAutoUpdateEnabled() bool
	IsPublicRegistryEnabled() bool
	CheckForDowngrade(latestVersion string) (bool, error)
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
		log.Info("updating version status according to custom image", "updater", updater.Name())
		err = updateVersionStatus(ctx, updater.Target(), customImage, reconciler.digestFunc, dockerCfg)
		return err
	}

	if !updater.IsAutoUpdateEnabled() {
		previousSource := updater.Target().Source
		emptyVersionStatus := dynatracev1beta1.VersionStatus{}
		if updater.Target() == nil || *updater.Target() == emptyVersionStatus {
			log.Info("initial status update in progress with no auto update", "updater", updater.Name())
		} else if previousSource == currentSource {
			log.Info("status updated skipped, due to no auto update", "updater", updater.Name())
			return nil
		}
	}

	if updater.IsPublicRegistryEnabled() {
		log.Info("updating version status according to public registry", "updater", updater.Name())
		var publicImage *dtclient.LatestImageInfo
		publicImage, err = updater.LatestImageInfo()
		if err != nil {
			log.Info("could not get public image", "updater", updater.Name())
			return err
		}
		isDowngrade, err := updater.CheckForDowngrade(publicImage.Tag)
		if err != nil {
			log.Info("could not determine if is downgrade for public registry", "updater", updater.Name())
			return err
		}
		if isDowngrade {
			return nil
		}

		err = updateVersionStatus(ctx, updater.Target(), publicImage.String(), reconciler.digestFunc, dockerCfg)
		if err != nil {
			log.Info("could not update version status according to the public registry", "updater", updater.Name())
			return err
		}
		return nil
	}

	log.Info("updating version status according to the tenant registry", "updater", updater.Name())
	err = updater.UseDefaults(ctx, dockerCfg)
	return err
}

func determineSource(updater versionStatusUpdater) dynatracev1beta1.VersionSource {
	if updater.CustomImage() != "" {
		return dynatracev1beta1.CustomImageVersionSource
	}
	if updater.IsPublicRegistryEnabled() {
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
	digestFunc ImageDigestFunc,
	dockerCfg *dockerconfig.DockerConfig,
) error {
	ref, err := reference.Parse(imageUri)
	if err != nil {
		return errors.WithMessage(err, "failed to parse image uri")
	}

	log.Info("updating image version info",
		"image", imageUri,
		"oldImageID", target.ImageID)

	if canonRef, ok := ref.(reference.Canonical); ok {
		target.ImageID = canonRef.String()
	} else if taggedRef, ok := ref.(reference.NamedTagged); ok {
		digest, err := digestFunc(ctx, imageUri, dockerCfg)
		if err != nil {
			target.ImageID = taggedRef.String()
			log.Error(err, "failed to get image digest, falling back to tag")
		} else {
			canonRef, err := reference.WithDigest(taggedRef, digest)
			if err != nil {
				target.ImageID = taggedRef.String()
				log.Error(err, "failed to create canonical image reference, falling back to tag")
			} else {
				target.ImageID = canonRef.String()
			}
		}
	} else {
		return errors.New(fmt.Sprintf("unsupported image reference: %s", imageUri))
	}

	log.Info("updated image version info",
		"newImageID", target.ImageID)

	// Version will be set elsewhere, as it differs between modes
	// unset is necessary so we have a consistent status
	target.Version = ""
	return nil
}
