package csiprovisioner

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

const (
	failedInstallAgentVersionEvent = "FailedInstallAgentVersion"
	installAgentVersionEvent       = "InstallAgentVersion"
)

var (
	log = logger.Factory.GetLogger("csi-provisioner")
)
