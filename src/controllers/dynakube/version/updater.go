package version

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type versionStatusUpdater interface {
	Name() string
	IsEnabled() bool
	Target() *status.VersionStatus

	CustomImage() string
	CustomVersion() string
	IsAutoUpdateEnabled() bool
	IsPublicRegistryEnabled() bool
	CheckForDowngrade(latestVersion string) (bool, error)
	LatestImageInfo() (*dtclient.LatestImageInfo, error)

	UseTenantRegistry(context.Context) error
}

func (reconciler *Reconciler) run(ctx context.Context, updater versionStatusUpdater) error {
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
		err = setImageIDWithDigest(ctx, reconciler.apiReader, reconciler.dynakube, updater.Target(), reconciler.versionFunc, customImage)
		return err
	}

	if !updater.IsAutoUpdateEnabled() {
		previousSource := updater.Target().Source
		emptyVersionStatus := status.VersionStatus{}
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
		if err != nil || isDowngrade {
			return err
		}

		err = setImageIDWithDigest(ctx, reconciler.apiReader, reconciler.dynakube, updater.Target(), reconciler.versionFunc, publicImage.String())
		if err != nil {
			log.Info("could not update version status according to the public registry", "updater", updater.Name())
			return err
		}
		return nil
	}

	log.Info("updating version status according to the tenant registry", "updater", updater.Name())
	err = updater.UseTenantRegistry(ctx)
	return err
}

func determineSource(updater versionStatusUpdater) status.VersionSource {
	if updater.CustomImage() != "" {
		return status.CustomImageVersionSource
	}
	if updater.IsPublicRegistryEnabled() {
		return status.PublicRegistryVersionSource
	}
	if updater.CustomVersion() != "" {
		return status.CustomVersionVersionSource
	}
	return status.TenantRegistryVersionSource
}

func setImageIDWithDigest( //nolint:revive
	ctx context.Context,
	apiReader client.Reader,
	dynakube *dynatracev1beta1.DynaKube,
	target *status.VersionStatus,
	imageVersionFunc ImageVersionFunc,
	imageUri string,
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
		registryClient := registry.NewClient()
		imageVersion, err := imageVersionFunc(ctx, apiReader, registryClient, dynakube, imageUri)
		if err != nil {
			log.Info("failed to determine image version")
			return err
		} else {
			canonRef, err := reference.WithDigest(taggedRef, imageVersion.Digest)
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

func updateVersionStatusForTenantRegistry( //nolint:revive
	ctx context.Context,
	apiReader client.Reader,
	dynakube *dynatracev1beta1.DynaKube,
	target *status.VersionStatus,
	imageVersionFunc ImageVersionFunc,
	imageUri string,
) error {
	ref, err := reference.Parse(imageUri)
	if err != nil {
		return errors.WithMessage(err, "failed to parse image uri")
	}

	log.Info("updating image version info for tenant registry image",
		"image", imageUri,
		"oldImageID", target.ImageID,
		"oldVersion", target.Version)

	if taggedRef, ok := ref.(reference.NamedTagged); ok {
		registryClient := registry.NewClient()
		imageVersion, err := imageVersionFunc(ctx, apiReader, registryClient, dynakube, imageUri)
		if err != nil {
			log.Info("failed to determine image version")
			return err
		}
		target.ImageID = taggedRef.String()
		target.Version = imageVersion.Version
	}

	log.Info("updated image version info for tenant registry image",
		"newImageID", target.ImageID,
		"newVersion", target.Version)

	return nil
}

func getTagFromImageID(imageID string) (string, error) {
	ref, err := reference.Parse(imageID)
	if err != nil {
		return "", err
	}
	taggedRef, ok := ref.(reference.NamedTagged)
	if !ok {
		return "", errors.New("no tag found to check for downgrade")
	}
	return taggedRef.Tag(), nil
}

func isDowngrade(updaterName, previousVersion, latestVersion string) (bool, error) {
	if previousVersion != "" {
		if downgrade, err := version.IsDowngrade(previousVersion, latestVersion); err != nil {
			return false, err
		} else if downgrade {
			log.Info("downgrade detected, which is not allowed in this configuration", "updater", updaterName, "from", previousVersion, "to", latestVersion)
			return true, err
		}
	}
	return false, nil
}
