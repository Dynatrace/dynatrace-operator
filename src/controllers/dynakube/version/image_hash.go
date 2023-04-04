package version

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// ImageDigestFunc can fetch image information from img
type ImageDigestFunc func(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (digest.Digest, error)

var _ ImageDigestFunc = GetImageDigest

// GetImageDigest fetches image information for imageName
func GetImageDigest(ctx context.Context, imageName string, dockerConfig *dockerconfig.DockerConfig) (digest.Digest, error) {
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

	return digest, nil
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
