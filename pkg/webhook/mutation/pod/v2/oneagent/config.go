package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("v2-pod-mutation-oneagent")
)

const (
	NoBootstrapperConfigReason = "NoBootstrapperConfig"

	oneAgentCodeModulesVolumeName       = "dynatrace-codemodules"
	oneAgentCodeModulesConfigVolumeName = "dynatrace-config"
	oneAgentCodeModulesConfigMountPath  = "/var/lib/dynatrace"

	bootstrapperSourceArgument = "source" // TODO import consts from bootstrapper as soon as >1.0.1 is released
	bootstrapperTargetArgument = "target"
)
