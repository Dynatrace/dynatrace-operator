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
const (
	VersionLabel = "com.dynatrace.build-version"
	TmpCAPath    = "/tmp/dynatrace-operator"
	TmpCAName    = "dynatraceCustomCA.crt"
)

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

	var ctx types.SystemContext

	if dockerConfig.SkipCertCheck {
		ctx.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
	}
	if dockerConfig.UseTrustedCerts {
		ctx.DockerCertPath = TmpCAPath
	}

	registry := strings.Split(dockerReference.Name(), "/")[0]

	for _, r := range []string{registry, "https://" + registry} {
		if creds, ok := dockerConfig.Auths[r]; ok {
			ctx.DockerAuthConfig = &types.DockerAuthConfig{Username: creds.Username, Password: creds.Password}
		}
	}

	return &ctx
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
