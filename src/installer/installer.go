package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type Installer interface {
	InstallAgent(targetDir string) (bool, error)
	UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error
}
