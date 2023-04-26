package version

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
)

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel = "com.dynatrace.build-version"
)

type ImageVersion struct {
	Version string
	Hash    string
}

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	systemContext := dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerConfig)

	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}
	defer closeImageSource(imageSource)

	imageManifest, _, err := imageSource.GetManifest(context.TODO(), nil)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	digest, err := manifest.Digest(imageManifest)
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	sourceImage, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	}

	inspectedImage, err := sourceImage.Inspect(context.TODO())
	if err != nil {
		return ImageVersion{}, errors.WithStack(err)
	} else if inspectedImage == nil {
		return ImageVersion{}, errors.Errorf("could not inspect image: '%s'", transportImageName)
	}

	return ImageVersion{
		Hash:    digest.Encoded(),
		Version: inspectedImage.Labels[VersionLabel], // empty if unset
	}, nil
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
