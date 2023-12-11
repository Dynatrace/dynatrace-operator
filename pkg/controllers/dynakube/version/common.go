package version

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ImageVersionReconciler interface {
	Reconcile(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error
}

// same for all
type getLatestImageFn func() (*dtclient.LatestImageInfo, error)

func getImageUri(dynakube *dynatracev1beta1.DynaKube, getLatestImage getLatestImageFn, defaultImage string) (string, error) {
	var imageUri string
	if dynakube.FeaturePublicRegistry() {
		publicImage, err := getLatestImage()
		if err != nil {
			log.Info("could not get public image", "updater", "activegate")
			return "", err
		}
		imageUri = publicImage.String()
	}
	if imageUri == "" {
		imageUri = defaultImage
	}
	return imageUri, nil
}

// same for all
func updateVersionWithImage(ctx context.Context, registryClient registry.ImageGetter, current status.VersionStatus, imageUri string) (status.VersionStatus, error) {
	ref, err := name.ParseReference(imageUri, name.WithDefaultTag(""))
	if err != nil {
		return status.VersionStatus{}, errors.WithMessage(err, "failed to parse image uri")
	}

	imageVersion, err := registryClient.GetImageVersion(ctx, imageUri)
	if err != nil {
		log.Info("failed to determine image version")
		return status.VersionStatus{}, err
	}
	// special treatment for tenant registry as the images are changed frequently due to config changes, so we cannot use the digest
	if current.Source == status.TenantRegistryVersionSource {
		if taggedRef, ok := ref.(name.Tag); ok {
			current.ImageID = taggedRef.String()
			current.Version = imageVersion.Version
			return current, nil
		} // todo: treat else branch here?
	}

	if digestRef, ok := ref.(name.Digest); ok {
		current.ImageID = digestRef.String()
		current.Version = imageVersion.Version
		return current, nil
	} else if taggedRef, ok := ref.(name.Tag); ok {
		if taggedRef.TagStr() == "" {
			return status.VersionStatus{}, errors.Errorf("unsupported image reference: %s", imageUri)
		}
		current.ImageID = registry.BuildImageIDWithTagAndDigest(taggedRef, imageVersion.Digest)
		current.Version = imageVersion.Version
		return current, nil
	} else {
		return status.VersionStatus{}, errors.Errorf("unsupported image reference: %s", imageUri)
	}
}

func isOutdated(logIt status.LogFn, timeProvider *timeprovider.Provider, dynakube *dynatracev1beta1.DynaKube, timestamp *v1.Time) bool {
	if !timeProvider.IsOutdated(timestamp, dynakube.FeatureApiRequestThreshold()) {
		logIt()
		return false
	}
	return true
}

func logSkipUpdateTimestampValidMessage(sourceName string) status.LogFn {
	return func() {
		log.Info("status timestamp still valid, skipping version status updater", "updater", sourceName)
	}
}

func InitialUpdateInProgress(s status.VersionStatus) bool {
	emptyVersionStatus := status.VersionStatus{}
	return s == emptyVersionStatus
}
