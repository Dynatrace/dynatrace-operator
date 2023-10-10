package version

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
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
		err = setImageIDWithDigest(ctx, updater.Target(), reconciler.registryClient, customImage)
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

		err = setImageIDWithDigest(ctx, updater.Target(), reconciler.registryClient, publicImage.String())
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

func setImageIDWithDigest(
	ctx context.Context,
	target *status.VersionStatus,
	registryClient registry.ImageGetter,
	imageUri string,
) error {
	ref, err := name.ParseReference(imageUri)
	if err != nil {
		return errors.WithMessage(err, "failed to parse image uri")
	}

	log.Info("updating image version info",
		"image", imageUri,
		"oldImageID", target.ImageID)

	if digestRef, ok := ref.(name.Digest); ok {
		target.ImageID = digestRef.String()
	} else if taggedRef, ok := ref.(name.Tag); ok {
		if taggedRef.TagStr() == name.DefaultTag {
			return errors.Errorf("unsupported image reference: %s", imageUri)
		}

		imageVersion, err := registryClient.GetImageVersion(ctx, imageUri)
		if err != nil {
			log.Info("failed to determine image version")
			return err
		}

		target.ImageID = registry.BuildImageIDWithTagAndDigest(taggedRef, imageVersion.Digest)
	} else {
		return errors.Errorf("unsupported image reference: %s", imageUri)
	}

	log.Info("updated image version info",
		"newImageID", target.ImageID)

	// Version will be set elsewhere, as it differs between modes
	// unset is necessary so we have a consistent status
	target.Version = ""
	return nil
}

func updateVersionStatusForTenantRegistry(
	ctx context.Context,
	target *status.VersionStatus,
	registryClient registry.ImageGetter,
	imageUri string,
) error {
	ref, err := name.ParseReference(imageUri)
	if err != nil {
		return errors.WithMessage(err, "failed to parse image uri")
	}

	log.Info("updating image version info for tenant registry image",
		"image", imageUri,
		"oldImageID", target.ImageID,
		"oldVersion", target.Version)

	if taggedRef, ok := ref.(name.Tag); ok {
		imageVersion, err := registryClient.GetImageVersion(ctx, imageUri)
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
	ref, err := name.ParseReference(imageID, name.WithDefaultTag(""))
	if err != nil {
		return "", err
	}

	var taggedRef name.Tag

	if digestRef, ok := ref.(name.Digest); ok {
		taggedStr := strings.TrimSuffix(digestRef.String(), registry.DigestDelimiter+digestRef.DigestStr())
		if taggedRef, err = name.NewTag(taggedStr, name.WithDefaultTag("")); err != nil {
			return "", err
		}
	} else if taggedRef, ok = ref.(name.Tag); !ok {
		return "", errors.New("no tag found to check for downgrade")
	}

	if taggedRef.TagStr() == "" {
		return "", errors.New("no tag found to check for downgrade")
	}

	return taggedRef.TagStr(), nil
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
