package dtversion

import (
	"fmt"
)

const VersionKey = "version"

type DockerLabelsChecker struct {
	image        string
	labels       map[string]string
	dockerConfig *DockerConfig
}

func NewDockerLabelsChecker(image string, labels map[string]string, dockerConfig *DockerConfig) *DockerLabelsChecker {
	return &DockerLabelsChecker{
		image:        image,
		labels:       labels,
		dockerConfig: dockerConfig,
	}
}

func (dockerLabelsChecker *DockerLabelsChecker) IsLatest() (bool, error) {
	versionLabel, hasVersionLabel := dockerLabelsChecker.labels[VersionKey]
	if !hasVersionLabel {
		return false, fmt.Errorf("key '%s' not found in given labels", VersionKey)
	}

	remoteVersionLabel, err := GetVersionLabel(dockerLabelsChecker.image, dockerLabelsChecker.dockerConfig)
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
