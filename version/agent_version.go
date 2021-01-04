package version

import (
	"github.com/go-logr/logr"
)

// Pre-production, adapt accordingly once images are released
var minSupportedAgentVersion = versionInfo{
	major:   1,
	minor:   203,
	release: 0,
}

func IsAgentVersionSupported(logger logr.Logger, versionString string) bool {
	if versionString == "" {
		// If version string is empty, latest agent image is used which is assumed to be supported
		return true
	}

	agentVersion, err := extractVersion(versionString)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	return IsSupportedAgentVersion(agentVersion)
}

func IsSupportedAgentVersion(agentVersion versionInfo) bool {
	comparison := compareVersionInfo(agentVersion, minSupportedAgentVersion)
	return comparison >= 0
}
