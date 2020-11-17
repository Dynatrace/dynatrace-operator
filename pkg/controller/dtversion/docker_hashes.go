package dtversion

import (
	"context"
	"fmt"
	"strings"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
)

// Deprecated: DockerLabelsChecker implements a preferred version check using image matchLabels
type DockerHashesChecker struct {
	currentImage   string
	currentImageId string
	dockerConfig   *DockerConfig
}

// Deprecated: DockerLabelsChecker implements a preferred version check using image matchLabels
func NewDockerHashesChecker(currentImage, currentImageId string, dockerConfig *DockerConfig) *DockerHashesChecker {
	return &DockerHashesChecker{
		currentImage:   currentImage,
		currentImageId: currentImageId,
		dockerConfig:   dockerConfig,
	}
}

func (dockerVersionChecker *DockerHashesChecker) IsLatest() (bool, error) {
	transportImageName := fmt.Sprintf("%s%s",
		"docker://",
		strings.TrimPrefix(
			dockerVersionChecker.currentImageId,
			"docker-pullable://"))

	latestReference, err := alltransports.ParseImageName("docker://" + dockerVersionChecker.currentImage)
	if err != nil {
		return false, err
	}

	latestDigest, err := dockerVersionChecker.getDigest(latestReference)
	if err != nil {
		return false, err
	}

	//Using ImageID instead of Image because ImageID contains digest of image that is used while Image only contains tag
	currentReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return false, err
	}

	currentDigest, err := dockerVersionChecker.getDigest(currentReference)
	if err != nil {
		return false, err
	}

	return currentDigest == latestDigest, nil
}

func (dockerVersionChecker *DockerHashesChecker) getDigest(ref types.ImageReference) (digest.Digest, error) {
	systemContext := MakeSystemContext(ref.DockerReference(), dockerVersionChecker.dockerConfig)
	imageSource, err := ref.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return "", err
	}
	defer closeImageSource(imageSource)

	imageManifest, _, err := imageSource.GetManifest(context.TODO(), nil)
	if err != nil {
		return "", err
	}

	imageDigest, err := manifest.Digest(imageManifest)
	if err != nil {
		return "", err
	}

	return imageDigest, nil
}

func closeImageSource(source types.ImageSource) {
	if source != nil {
		// Swallow error
		_ = source.Close()
	}
}
