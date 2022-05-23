package oneagent_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.oneagent")
)

const (
	preloadEnvVarName           = "LD_PRELOAD"
	networkZoneEnvVarName       = "DT_NETWORK_ZONE"
	proxyEnvVarName             = "DT_PROXY"
	dynatraceMetadataEnvVarName = "DT_DEPLOYMENT_METADATA"
	initialConnectRetryEnvVarName = "DT_INITIAL_CONNECT_RETRY_MS"

	oneAgentBinVolumeName   = "oneagent-bin"
	oneAgentShareVolumeName = "oneagent-share"

	injectionConfigVolumeName = "injection-config"

	provisionedVolumeMode = "provisioned"
	installerVolumeMode   = "installer"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"
	customCertFileName     = "custom.pem"
	proxySecretKey         = "proxy"

	preloadPath       = "/etc/ld.so.preload"
	preloadSubPath    = "ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"
	libAgentProcPath  = "/agent/lib64/liboneagentproc.so"
)
