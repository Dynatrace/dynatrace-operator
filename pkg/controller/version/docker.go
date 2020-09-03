package version

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"
	"strings"
)

type DockerVersionChecker struct {
	currentImage   string
	currentImageId string
	dockerConfig   *parser.DockerConfig
}

func NewDockerVersionChecker(currentImage, currentImageId string, dockerConfig *parser.DockerConfig) *DockerVersionChecker {
	return &DockerVersionChecker{
		currentImage:   currentImage,
		currentImageId: currentImageId,
		dockerConfig:   dockerConfig,
	}
}

func (dockerVersionChecker *DockerVersionChecker) IsLatest() (bool, error) {
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
	//reference, err := name.ParseReference(strings.TrimPrefix(dockerVersionChecker.currentImageId, "docker-pullable://"))
	if err != nil {
		return false, err
	}

	currentDigest, err := dockerVersionChecker.getDigest(currentReference)
	if err != nil {
		return false, err
	}

	return currentDigest == latestDigest, nil
}

func (dockerVersionChecker *DockerVersionChecker) getDigest(ref types.ImageReference) (digest.Digest, error) {
	systemContext := dockerVersionChecker.makeSystemContext(ref.DockerReference())
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

func (dockerVersionChecker *DockerVersionChecker) makeSystemContext(dockerReference reference.Named) *types.SystemContext {
	if dockerReference == nil || dockerVersionChecker.dockerConfig == nil {
		return &types.SystemContext{}
	}

	registryName := strings.Split(dockerReference.Name(), "/")[0]
	credentials, hasCredentials := dockerVersionChecker.dockerConfig.Auths[registryName]

	if !hasCredentials {
		registryURL := "https://" + registryName
		credentials, hasCredentials = dockerVersionChecker.dockerConfig.Auths[registryURL]
		if !hasCredentials {
			return &types.SystemContext{}
		}
	}

	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: credentials.Username,
			Password: credentials.Password,
		}}

}
