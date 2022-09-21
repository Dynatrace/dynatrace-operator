package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type Installer interface {
	InstallAgent(targetDir string) (bool, error)
	UpdateProcessModuleConfig(configDir string, agentInstallDir string, processModuleConfig *dtclient.ProcessModuleConfig) error
}
