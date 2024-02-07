package csiprovisioner

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

const (
	failedInstallAgentVersionEvent = "FailedInstallAgentVersion"
	installAgentVersionEvent       = "InstallAgentVersion"
)

var (
	log = logger.Get().WithName("csi-provisioner")
)
