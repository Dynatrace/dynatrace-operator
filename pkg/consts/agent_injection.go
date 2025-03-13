package consts

type InstallMode string

const (
	AgentNoHostTenant                  = "-"
	AgentContainerConfFilenameTemplate = "container_%s.conf"
	AgentInitSecretName                = "dynatrace-dynakube-config"
	AgentInitSecretConfigField         = "config"

	BootstrapperInitSecretName = "dynatrace-bootstrapper-config"

	LdPreloadFilename = "ld.so.preload"
	LibAgentProcPath  = "/agent/lib64/liboneagentproc.so"

	AgentCurlOptionsFileName = "curl_options.conf"

	AgentInstallerUrlEnv     = "INSTALLER_URL"
	AgentInstallerFlavorEnv  = "FLAVOR"
	AgentInstallerTechEnv    = "TECHNOLOGIES"
	AgentInstallerVersionEnv = "VERSION"

	AgentInstallPathEnv = "INSTALLPATH"

	AgentInjectedEnv = "ONEAGENT_INJECTED"

	AgentBinDirMount      = "/mnt/bin"
	AgentShareDirMount    = "/mnt/share"
	AgentConfigDirMount   = "/mnt/config"
	AgentConfInitDirMount = "/mnt/agent-conf"
	AgentCodeModuleSource = "/opt/dynatrace/oneagent"

	TrustedCAsInitSecretField    = "trustedcas"
	ActiveGateCAsInitSecretField = "agcerts"
	CustomCertsFileName          = "custom.pem"
	CustomProxyCertsFileName     = "custom_proxy.pem"
)
