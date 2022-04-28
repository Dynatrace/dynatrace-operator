package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

type Installer interface {
	InstallAgent(targetDir string) error
	UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error
}
