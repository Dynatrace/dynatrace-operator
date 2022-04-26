package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
)

type Installer interface {
	InstallAgent(targetDir string) error
	UpdateProcessModuleConfig(targetDir string, processModuleConfig *types.ProcessModuleConfig) error
}
