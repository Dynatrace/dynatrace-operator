package csiprovisioner

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	failedInstallAgentVersionEvent = "FailedInstallAgentVersion"
	installAgentVersionEvent       = "InstallAgentVersion"
)

var (
	log = logd.Get().WithName("csi-provisioner")
)
