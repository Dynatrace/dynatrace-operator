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

type StatusUpdater interface {
	Name() string
	IsEnabled() bool
	Target() *status.VersionStatus

	CustomImage() string
	CustomVersion() string
	IsAutoUpdateEnabled() bool
	IsPublicRegistryEnabled() bool
	CheckForDowngrade(latestVersion string) (bool, error)
	ValidateStatus() error
	LatestImageInfo(ctx context.Context) (*dtclient.LatestImageInfo, error)

	UseTenantRegistry(context.Context) error
}

func (r *reconciler) run(ctx context.Context, updater StatusUpdater) error {
	currentSource := determineSource(updater)

	var err error

	defer func() {
		if err == nil {
			updater.Target().LastProbeTimestamp = r.timeProvider.Now()
			updater.Target().Source = currentSource
		}
	}()

	customImage := updater.CustomImage()
	if customImage != "" {
		log.Info("updating version status according to custom image", "updater", updater.Name())
		setImageIDToCustomImage(updater.Target(), customImage)

		return nil
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
		err = r.processPublicRegistry(ctx, updater)
		if err != nil {
			return err
		}

		return updater.ValidateStatus()
	}

	log.Info("updating version status according to the tenant registry", "updater", updater.Name())

	err = updater.UseTenantRegistry(ctx)
	if err != nil {
		return err
	}

	return updater.ValidateStatus()
}

func (r *reconciler) processPublicRegistry(ctx context.Context, updater StatusUpdater) error {
	log.Info("updating version status according to public registry", "updater", updater.Name())

	var publicImage *dtclient.LatestImageInfo

	publicImage, err := updater.LatestImageInfo(ctx)
	if err != nil {
		log.Info("could not get public image", "updater", updater.Name())

		return err
	}

	isDowngrade, err := updater.CheckForDowngrade(publicImage.Tag)
	if err != nil || isDowngrade {
		return err
	}

	setImageFromImageInfo(updater.Target(), *publicImage)

	return nil
}

func determineSource(updater StatusUpdater) status.VersionSource {
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

func setImageIDToCustomImage(
	target *status.VersionStatus,
	imageUri string,
) {
	log.Info("updating image version info",
		"image", imageUri,
		"oldImageID", target.ImageID)

	target.ImageID = imageUri
	target.Version = string(status.CustomImageVersionSource)

	log.Info("updated image version info",
		"newImageID", target.ImageID)
}

func setImageFromImageInfo(
	target *status.VersionStatus,
	imageInfo dtclient.LatestImageInfo,
) {
	imageUri := imageInfo.String()
	log.Info("updating image version info",
		"image", imageInfo.String(),
		"oldImageID", target.ImageID)

	target.Version = imageInfo.Tag
	target.ImageID = imageUri

	log.Info("updated image version info",
		"newImageID", target.ImageID)
}

func updateVersionStatusForTenantRegistry(
	target *status.VersionStatus,
	imageUri string,
	latestVersion string,
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
		target.ImageID = taggedRef.String()
		target.Version = latestVersion
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
