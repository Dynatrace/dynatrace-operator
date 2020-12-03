package dtversion

import (
	"context"
	"fmt"
	"strings"

	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
)

// VersionLabel is the name of the label used on ActiveGate-provided images.
const VersionLabel = "com.dynatrace.build-version"

// ImageVersion includes information for a given image. Version can be empty if the corresponding label isn't set.
type ImageVersion struct {
	Version string
	Hash    string
}

// ImageVersionProvider can fetch image information from img
type ImageVersionProvider func(img string, dockerConfig *DockerConfig) (ImageVersion, error)

var _ ImageVersionProvider = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion(imageName string, dockerConfig *DockerConfig) (ImageVersion, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return ImageVersion{}, err
	}

	systemContext := MakeSystemContext(imageReference.DockerReference(), dockerConfig)

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

// MakeSystemContext returns a SystemConfig for the given image and Dockerconfig.
func MakeSystemContext(dockerReference reference.Named, dockerConfig *DockerConfig) *types.SystemContext {
	if dockerReference == nil || dockerConfig == nil {
		return &types.SystemContext{}
	}

	registryName := strings.Split(dockerReference.Name(), "/")[0]
	credentials, hasCredentials := dockerConfig.Auths[registryName]

	if !hasCredentials {
		registryURL := "https://" + registryName
		credentials, hasCredentials = dockerConfig.Auths[registryURL]
		if !hasCredentials {
			return &types.SystemContext{}
		}
	}

	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: credentials.Username,
			Password: credentials.Password,
		},
	}
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
