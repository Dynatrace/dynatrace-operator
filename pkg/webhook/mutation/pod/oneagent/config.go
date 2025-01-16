package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("oneagent-pod-mutation")
)

const (
	preloadEnv           = "LD_PRELOAD"
	networkZoneEnv       = "DT_NETWORK_ZONE"
	dynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	releaseVersionEnv      = "DT_RELEASE_VERSION"
	releaseProductEnv      = "DT_RELEASE_PRODUCT"
	releaseStageEnv        = "DT_RELEASE_STAGE"
	releaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	EmptyConnectionInfoReason = "EmptyConnectionInfo"
	UnknownCodeModuleReason   = "UnknownCodeModule"
	EmptyTenantUUIDReason     = "EmptyTenantUUID"

	OneAgentBinVolumeName = "oneagent-bin"
	preloadPath           = "/etc/ld.so.preload"

	// readonly CSI
	oneagentConfVolumeName = "oneagent-agent-conf"
	OneAgentConfMountPath  = "/opt/dynatrace/oneagent-paas/agent/conf"

	oneagentDataStorageVolumeName = "oneagent-data-storage"
	oneagentDataStorageMountPath  = "/opt/dynatrace/oneagent-paas/datastorage"

	oneagentLogVolumeName = "oneagent-log"
	oneagentLogMountPath  = "/opt/dynatrace/oneagent-paas/log"
)
