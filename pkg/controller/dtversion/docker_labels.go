package dtversion

import (
	"fmt"
)

const VersionKey = "version"

type DockerLabelsChecker struct {
	image                       string
	labels                      map[string]string
	dockerConfig                *DockerConfig
	imageInformationConstructor func(string, *DockerConfig) ImageInformation
}

func NewDockerLabelsChecker(image string, labels map[string]string, dockerConfig *DockerConfig) *DockerLabelsChecker {
	return &DockerLabelsChecker{
		image:                       image,
		labels:                      labels,
		dockerConfig:                dockerConfig,
		imageInformationConstructor: NewPodImageInformation,
	}
}

func (dockerLabelsChecker *DockerLabelsChecker) IsLatest() (bool, error) {
	versionLabel, hasVersionLabel := dockerLabelsChecker.labels[VersionKey]
	if !hasVersionLabel {
		return false, fmt.Errorf("key '%s' not found in given matchLabels", VersionKey)
	}

	remoteVersionLabel, err := dockerLabelsChecker.
		imageInformationConstructor(
			dockerLabelsChecker.image, dockerLabelsChecker.dockerConfig).
		GetVersionLabel()
	if err != nil {
		return false, err
	}

	localVersion, err := ExtractVersion(versionLabel)
	if err != nil {
		return false, err
	}

	remoteVersion, err := ExtractVersion(remoteVersionLabel)
	if err != nil {
		return false, err
	}

	// Return true if local version is not equal to the remote version
	return CompareVersionInfo(localVersion, remoteVersion) != 0, nil
}
