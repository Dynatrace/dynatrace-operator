package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-bootstrapper-oneagent")
)

const (
	preloadEnv = "LD_PRELOAD"

	releaseVersionEnv      = "DT_RELEASE_VERSION"
	releaseProductEnv      = "DT_RELEASE_PRODUCT"
	releaseStageEnv        = "DT_RELEASE_STAGE"
	releaseBuildVersionEnv = "DT_RELEASE_BUILD_VERSION"

	emptyConnectionInfoReason = "EmptyConnectionInfo"
	emptyTenantUUIDReason     = "EmptyTenantUUID"

	oneAgentCodeModulesVolumeName       = "dynatrace-codemodules"
	oneAgentCodeModulesConfigVolumeName = "dynatrace-config"
	oneAgentCodeModulesConfigMountPath  = "/var/lib/dynatrace"

	bootstrapperSourceArgument     = "source"
	bootstrapperTargetArgument     = "target"
	bootstrapperTechnologyArgument = "technology"
)
