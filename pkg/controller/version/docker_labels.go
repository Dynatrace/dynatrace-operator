package version

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/containers/image/v5/image"
	"github.com/containers/image/v5/transports/alltransports"
)

const versionKey = "version"

type DockerLabelsChecker struct {
	image        string
	labels       map[string]string
	dockerConfig *parser.DockerConfig
}

func NewDockerLabelsChecker(image string, labels map[string]string, dockerConfig *parser.DockerConfig) *DockerLabelsChecker {
	return &DockerLabelsChecker{
		image:        image,
		labels:       labels,
		dockerConfig: dockerConfig,
	}
}

func (dockerLabelsChecker *DockerLabelsChecker) IsLatest() (bool, error) {
	versionLabel, hasVersionLabel := dockerLabelsChecker.labels[versionKey]
	if !hasVersionLabel {
		return false, fmt.Errorf("key '%s' not found in given labels", versionKey)
	}

	transportImageName := fmt.Sprintf("docker://%s", dockerLabelsChecker.image)

	imageReference, err := alltransports.ParseImageName(transportImageName)
	if err != nil {
		return false, err
	}

	systemContext := makeSystemContext(imageReference.DockerReference(), dockerLabelsChecker.dockerConfig)
	imageSource, err := imageReference.NewImageSource(context.TODO(), systemContext)
	if err != nil {
		return false, err
	}
	defer closeImageSource(imageSource)

	img, err := image.FromUnparsedImage(context.TODO(), systemContext, image.UnparsedInstance(imageSource, nil))
	if err != nil {
		return false, err
	}
	if img == nil {
		return false, fmt.Errorf("could not find image: '%s'", transportImageName)
	}

	inspectedImg, err := img.Inspect(context.TODO())
	if err != nil {
		return false, err
	}
	if inspectedImg == nil {
		return false, fmt.Errorf("could not inspect image: '%s'", transportImageName)
	}

	remoteVersionLabel, hasRemoteVersionLabel := inspectedImg.Labels[versionKey]
	if !hasRemoteVersionLabel {
		return false, fmt.Errorf("remote does not have key '%s' in labels", versionKey)
	}

	localVersion, err := extractVersion(versionLabel)
	if err != nil {
		return false, err
	}

	remoteVersion, err := extractVersion(remoteVersionLabel)
	if err != nil {
		return false, err
	}

	// Return true if local version is equal or greater to the remote version
	return compareVersionInfo(localVersion, remoteVersion) >= 0, nil
}
