package consts

type InstallMode string

const (
	AgentNoHostTenant                  = "-"
	AgentContainerConfFilenameTemplate = "container_%s.conf"
	AgentInitSecretName                = "dynatrace-dynakube-config"
	AgentInitSecretConfigField         = "config"
	AgentInitSecretTrustedCAsField     = "trustedCAs"

	LdPreloadFilename = "ld.so.preload"
	LibAgentProcPath  = "/agent/lib64/liboneagentproc.so"

	AgentCurlOptionsFileName = "curl_options.conf"

	AgentInstallerUrlEnv     = "INSTALLER_URL"
	AgentInstallerFlavorEnv  = "FLAVOR"
	AgentInstallerTechEnv    = "TECHNOLOGIES"
	AgentInstallerVersionEnv = "VERSION"

	AgentInstallPathEnv            = "INSTALLPATH"
	AgentContainerCountEnv         = "CONTAINERS_COUNT"
	AgentContainerNameEnvTemplate  = "CONTAINER_%d_NAME"
	AgentContainerImageEnvTemplate = "CONTAINER_%d_IMAGE"

	AgentInjectedEnv = "ONEAGENT_INJECTED"

	AgentBinDirMount      = "/mnt/bin"
	AgentShareDirMount    = "/mnt/share"
	AgentConfigDirMount   = "/mnt/config"
	AgentConfInitDirMount = "/mnt/agent-conf"
)
