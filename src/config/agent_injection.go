package config

type InstallMode string

const (
	AgentNoHostTenant                  = "-"
	AgentContainerConfFilenameTemplate = "container_%s.conf"
	AgentInitSecretName                = "dynatrace-dynakube-config"
	AgentInitSecretConfigField         = "config"

	LdPreloadFilename = "ld.so.preload"
	LibAgentProcPath  = "/agent/lib64/liboneagentproc.so"

	AgentCurlOptionsFileName = "curl_options.conf"

	AgentInstallerMode InstallMode = "installer"
	AgentCsiMode       InstallMode = "provisioned"

	AgentInstallModeEnv      = "MODE"
	AgentInstallerUrlEnv     = "INSTALLER_URL"
	AgentInstallerFlavorEnv  = "FLAVOR"
	AgentInstallerTechEnv    = "TECHNOLOGIES"
	AgentInstallerVersionEnv = "VERSION"

	AgentInstallPathEnv            = "INSTALLPATH"
	AgentContainerCountEnv         = "CONTAINERS_COUNT"
	AgentContainerNameEnvTemplate  = "CONTAINER_%d_NAME"
	AgentContainerImageEnvTemplate = "CONTAINER_%d_IMAGE"

	AgentInjectedEnv = "ONEAGENT_INJECTED"
	AgentReadonlyCSI = "CSI_VOLUME_READONLY"

	AgentBinDirMount    = "/mnt/bin"
	AgentShareDirMount  = "/mnt/share"
	AgentConfigDirMount = "/mnt/config"
	AgentConfDirMount   = "/mnt/bin/agent-conf"
)
