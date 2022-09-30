package oneagent_mutation

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.pod.oneagent")

	envToBuildFieldPathMap = map[string]string{
		"DT_RELEASE_VERSION": "metadata.labels['app.kubernetes.io/version']",
		"DT_RELEASE_PRODUCT": "metadata.labels['app.kubernetes.io/part-of`]",
	}
)

const (
	preloadEnv           = "LD_PRELOAD"
	networkZoneEnv       = "DT_NETWORK_ZONE"
	proxyEnv             = "DT_PROXY"
	dynatraceMetadataEnv = "DT_DEPLOYMENT_METADATA"

	OneAgentBinVolumeName     = "oneagent-bin"
	oneAgentShareVolumeName   = "oneagent-share"
	injectionConfigVolumeName = "injection-config"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"
	customCertFileName     = "custom.pem"

	preloadPath       = "/etc/ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"
)
