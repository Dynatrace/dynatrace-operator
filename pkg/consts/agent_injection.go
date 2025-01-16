package consts

type InstallMode string

const (
	AgentNoHostTenant          = "-"
	AgentInitSecretName        = "dynatrace-dynakube-config"
	AgentInitSecretConfigField = "config"

	LdPreloadFilename = "ld.so.preload"
	LibAgentProcPath  = "/agent/lib64/liboneagentproc.so"

	AgentInstallerUrlEnv     = "INSTALLER_URL"
	AgentInstallerFlavorEnv  = "FLAVOR"
	AgentInstallerTechEnv    = "TECHNOLOGIES"
	AgentInstallerVersionEnv = "VERSION"

	AgentInstallPathEnv = "INSTALLPATH"

	AgentInjectedEnv = "ONEAGENT_INJECTED"

	AgentBinDirMount      = "/mnt/bin"
	AgentConfInitDirMount = "/mnt/agent-conf"

	AgentSubDirName          = "oneagent"
	AgentCustomKeysSubDir    = "agent/customkeys"
	AgentCurlOptionsFileName = "curl_options.conf"
	CustomCertsFileName      = "custom.pem"
	CustomProxyCertsFileName = "custom_proxy.pem"

	AgentContainerConfSubDir = "agent/config/container.conf"

	TrustedCAsInitSecretField    = "trustedcas"
	ActiveGateCAsInitSecretField = "agcerts"
)
