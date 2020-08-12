package version

import (
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"regexp"
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
	regex := regexp.MustCompile("(^docker-pullable:\\/\\/|\\:.*$|\\@sha256.*$)")
	latestImageName := regex.ReplaceAllString(dockerVersionChecker.currentImage, "") + ":latest"

	//Using ImageID instead of Image because ImageID contains digest of image that is used while Image only contains tag
	reference, err := name.ParseReference(strings.TrimPrefix(dockerVersionChecker.currentImageId, "docker-pullable://"))
	if err != nil {
		return false, err
	}

	latestReference, err := name.ParseReference(latestImageName)
	if err != nil {
		return false, err
	}

	registryURL := "https://" + reference.Context().RegistryStr()
	authOption := getAuthOption(dockerVersionChecker.dockerConfig, registryURL)

	latestDigest, err := getDigest(latestReference, authOption)
	if err != nil {
		return false, err
	}

	currentDigest, err := getDigest(reference, authOption)
	if err != nil {
		return false, err
	}

	return currentDigest == latestDigest, nil
}

func getDigest(reference name.Reference, authOption remote.Option) (string, error) {
	img, err := remote.Image(reference, authOption)
	if err != nil {
		return "", err
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return digest.Hex, nil
}

func getAuthOption(dockerConfig *parser.DockerConfig, registryURL string) remote.Option {
	if dockerConfig == nil {
		return remote.WithAuthFromKeychain(authn.DefaultKeychain)
	}

	credentials, hasCredentials := dockerConfig.Auths[registryURL]
	if !hasCredentials {
		return remote.WithAuthFromKeychain(authn.DefaultKeychain)
	}

	return remote.WithAuth(authn.FromConfig(authn.AuthConfig{
		Username: credentials.Username,
		Password: credentials.Password,
	}))
}
