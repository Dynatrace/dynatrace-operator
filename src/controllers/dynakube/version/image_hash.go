package version

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
)

// ImageHashFunc can fetch image information from img
type ImageHashFunc func(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (string, error)

var _ ImageHashFunc = GetImageHash

// GetImageHash fetches image information for imageName
func GetImageHash(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (string, error) {
	transportImageName := fmt.Sprintf("docker://%s", imageName)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	systemContext := dockerconfig.MakeSystemContext(imageReference.DockerReference(), dockerConfig)

	imageSource, err := imageReference.NewImageSource(ctx, systemContext)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer closeImageSource(imageSource)

	imageManifest, _, err := imageSource.GetManifest(ctx, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}

	digest, err := manifest.Digest(imageManifest)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return digest.String(), nil
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
