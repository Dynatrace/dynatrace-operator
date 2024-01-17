package oneagent_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var (
	log = logger.Get().WithName("oneagent-pod-mutation")
)

const (
	preloadEnv           = "LD_PRELOAD"
	networkZoneEnv       = "DT_NETWORK_ZONE"
	dynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	releaseVersionEnv      = "DT_RELEASE_VERSION"
	releaseProductEnv      = "DT_RELEASE_PRODUCT"
	releaseStageEnv        = "DT_RELEASE_STAGE"
	releaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	OneAgentBinVolumeName        = "oneagent-bin"
	oneAgentShareVolumeName      = "oneagent-share"
	oneAgentTrustedCAsVolumeName = "oneagent-trustedcas"
	injectionConfigVolumeName    = "injection-config"

	oneAgentCustomKeysPath  = "/var/lib/dynatrace/oneagent/agent/customkeys"
	customCertFileName      = "custom.pem"
	customProxyCertFileName = "custom-proxy.pem"

	preloadPath       = "/etc/ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"

	// readonly CSI
	oneagentConfVolumeName = "oneagent-agent-conf"
	OneAgentConfMountPath  = "/opt/dynatrace/oneagent-paas/agent/conf"

	oneagentDataStorageVolumeName = "oneagent-data-storage"
	oneagentDataStorageMountPath  = "/opt/dynatrace/oneagent-paas/datastorage"

	oneagentLogVolumeName = "oneagent-log"
	oneagentLogMountPath  = "/opt/dynatrace/oneagent-paas/log"
)
