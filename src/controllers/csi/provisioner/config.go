package csiprovisioner

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	failedInstallAgentVersionEvent = "FailedInstallAgentVersion"
	installAgentVersionEvent       = "InstallAgentVersion"
)

var (
	log = logger.NewDTLogger().WithName("csi-provisioner")
)
