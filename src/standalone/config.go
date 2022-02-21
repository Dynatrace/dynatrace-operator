package standalone

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

type InstallMode string

const (
	NoHostTenant                  = "-"
	ContainerConfFilenameTemplate = "container_%s.conf"
	SecretConfigFieldName         = "config"

	enrichmentFilenameTemplate = "dt_metadata.%s"
	ldPreloadFilename          = "ld.so.preload"

	// Env Vars
	InstallerMode InstallMode = "installer"
	CsiMode       InstallMode = "csi"

	ModeEnv         = "MODE"
	CanFailEnv      = "FAILURE_POLICY"
	InstallerUrlEnv = "INSTALLER_URL"

	InstallerFlavorEnv = "FLAVOR"
	InstallerTechEnv   = "TECHNOLOGIES"
	InstallerArchEnv   = "ARCH"

	K8NodeNameEnv    = "K8S_NODE_NAME"
	K8PodNameEnv     = "K8S_PODNAME"
	K8PodUIDEnv      = "K8S_PODUID"
	K8BasePodNameEnv = "K8S_BASEPODNAME"
	K8NamespaceEnv   = "K8S_NAMESPACE"

	WorkloadKindEnv = "DT_WORKLOAD_KIND"
	WorkloadNameEnv = "DT_WORKLOAD_NAME"

	InstallPathEnv            = "INSTALLPATH"
	ContainerCountEnv         = "CONTAINERS_COUNT"
	ContainerNameEnvTemplate  = "CONTAINER_%d_NAME"
	ContainerImageEnvTemplate = "CONTAINER_%d_IMAGE"

	OneAgentInjectedEnv   = "ONEAGENT_INJECTED"
	DataIngestInjectedEnv = "DATA_INGEST_INJECTED"
)

var (
	log = logger.NewDTLogger().WithName("standalone-init")

	// Mount Path
	BinDirMount    = filepath.Join("mnt", "bin")
	ShareDirMount  = filepath.Join("mnt", "share")
	ConfigDirMount = filepath.Join("mnt", "config")

	EnrichmentPath = filepath.Join("var", "lib", "dynatrace", "enrichment")
)
