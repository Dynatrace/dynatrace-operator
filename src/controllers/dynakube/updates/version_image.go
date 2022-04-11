package updates

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

const (
	// VersionLabel is the name of the label used on ActiveGate-provided images.
	VersionLabel = "com.dynatrace.build-version"
)

// ImageVersion includes information for a given image. Version can be empty if the corresponding label isn't set.
type ImageVersion struct {
	Version string
	Hash    string
}

// ImageVersionProvider can fetch image information from img
type ImageVersionProvider func(img string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error)

var _ ImageVersionProvider = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(imageName string, dockerConfig *dockerconfig.DockerConfig) (ImageVersion, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return ImageVersion{}, err
	}

	systemContext := dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerConfig)

	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return ImageVersion{}, err
	}
	defer closeImageSource(imageSource)

	imageManifest, _, err := imageSource.GetManifest(context.TODO(), nil)
	if err != nil {
		return ImageVersion{}, err
	}

	digest, err := manifest.Digest(imageManifest)
	if err != nil {
		return ImageVersion{}, err
	}

	image, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return ImageVersion{}, err
	} else if image == nil {
		return ImageVersion{}, fmt.Errorf("could not find image: '%s'", transportImageName)
	}

	inspectedImage, err := image.Inspect(context.TODO())
	if err != nil {
		return ImageVersion{}, err
	} else if inspectedImage == nil {
		return ImageVersion{}, fmt.Errorf("could not inspect image: '%s'", transportImageName)
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
