package oneagent_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.pod.oneagent")
)

const (
	preloadEnvVarName           = "LD_PRELOAD"
	networkZoneEnvVarName       = "DT_NETWORK_ZONE"
	proxyEnvVarName             = "DT_PROXY"
	dynatraceMetadataEnvVarName = "DT_DEPLOYMENT_METADATA"

	oneAgentBinVolumeName     = "oneagent-bin"
	oneAgentShareVolumeName   = "oneagent-share"
	injectionConfigVolumeName = "injection-config"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"
	customCertFileName     = "custom.pem"

	preloadPath       = "/etc/ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"
)
