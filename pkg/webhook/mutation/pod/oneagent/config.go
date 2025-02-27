package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("oneagent-pod-mutation")
)

const (
	networkZoneEnv       = "DT_NETWORK_ZONE"
	dynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	EmptyConnectionInfoReason = "EmptyConnectionInfo"
	UnknownCodeModuleReason   = "UnknownCodeModule"
	EmptyTenantUUIDReason     = "EmptyTenantUUID"

	OneAgentBinVolumeName     = "oneagent-bin"
	oneAgentShareVolumeName   = "oneagent-share"
	injectionConfigVolumeName = "injection-config"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"

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
