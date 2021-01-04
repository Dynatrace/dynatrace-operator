package version

import (
	"github.com/go-logr/logr"
)

var minSupportedClusterVersion = versionInfo{
	major:   1,
	minor:   205,
	release: 0,
}

func IsRemoteClusterVersionSupported(logger logr.Logger, clusterVersion string) bool {
	remoteVersion, err := extractVersion(clusterVersion)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	return isSupportedClusterVersion(remoteVersion)
}

func isSupportedClusterVersion(clusterVersion versionInfo) bool {
	comparison := compareVersionInfo(clusterVersion, minSupportedClusterVersion)
	return comparison >= 0
}
