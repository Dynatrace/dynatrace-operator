package oneagent_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.Factory.GetLogger("mutation-oneagent")
)

const (
	preloadEnv           = "LD_PRELOAD"
	networkZoneEnv       = "DT_NETWORK_ZONE"
	proxyEnv             = "DT_PROXY"
	dynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	releaseVersionEnv      = "DT_RELEASE_VERSION"
	releaseProductEnv      = "DT_RELEASE_PRODUCT"
	releaseStageEnv        = "DT_RELEASE_STAGE"
	releaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	OneAgentBinVolumeName     = "oneagent-bin"
	oneAgentShareVolumeName   = "oneagent-share"
	injectionConfigVolumeName = "injection-config"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"
	customCertFileName     = "custom.pem"

	preloadPath       = "/etc/ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"
)
