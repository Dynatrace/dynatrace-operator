package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/mock"
)

type InstallerMock struct {
	mock.Mock
}

var _ Installer = &InstallerMock{}

func (mock *InstallerMock) InstallAgent(targetDir string) (bool, error) {
	args := mock.Called(targetDir)
	return args.Bool(0), args.Error(1)
}

func (mock *InstallerMock) UpdateProcessModuleConfig(configDir string, agentInstallerDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	args := mock.Called(configDir, agentInstallerDir, processModuleConfig)
	return args.Error(0)
}
