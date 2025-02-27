package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-bootstrapper-oneagent")
)

const (
	emptyConnectionInfoReason = "EmptyConnectionInfo"
	emptyTenantUUIDReason     = "EmptyTenantUUID"

	oneAgentCodeModulesVolumeName       = "dynatrace-codemodules"
	oneAgentCodeModulesConfigVolumeName = "dynatrace-config"
	oneAgentCodeModulesConfigMountPath  = "/var/lib/dynatrace"

	bootstrapperSourceArgument     = "source"
	bootstrapperTargetArgument     = "target"
	bootstrapperTechnologyArgument = "technology"
)
