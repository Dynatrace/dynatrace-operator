package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("v1-pod-mutation-oneagent")
)

const (
	OneAgentBinVolumeName     = "oneagent-bin"
	oneAgentShareVolumeName   = "oneagent-share"
	injectionConfigVolumeName = "injection-config"

	oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"

	preloadPath       = "/etc/ld.so.preload"
	containerConfPath = "/var/lib/dynatrace/oneagent/agent/config/container.conf"

	// readonly CSI
	oneagentConfVolumeName = "oneagent-agent-conf"
	OneAgentConfMountPath  = "/opt/dynatrace/oneagent-paas/agent/conf"

	oneagentDataStorageVolumeName = "oneagent-data-storage"
	oneagentDataStorageMountPath  = "/opt/dynatrace/oneagent-paas/datastorage"

	oneagentLogVolumeName = "oneagent-log"
	oneagentLogMountPath  = "/opt/dynatrace/oneagent-paas/log"

	// AnnotationFlavor can be set on a Pod to configure which code modules flavor to download. It's set to "default"
	// if not set.
	AnnotationFlavor = "oneagent.dynatrace.com/flavor"

	// AnnotationInstallerUrl can be set on a Pod to configure the installer url for downloading the agent
	// defaults to the PaaS installer download url of your tenant
	AnnotationInstallerUrl = "oneagent.dynatrace.com/installer-url"
)
