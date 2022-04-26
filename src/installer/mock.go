package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient/types"
	"github.com/stretchr/testify/mock"
)

type InstallerMock struct {
	mock.Mock
}

var _ Installer = &InstallerMock{}

func (mock *InstallerMock) InstallAgent(targetDir string) error {
	args := mock.Called(targetDir)
	return args.Error(0)
}

func (mock *InstallerMock) UpdateProcessModuleConfig(targetDir string, processModuleConfig *types.ProcessModuleConfig) error {
	args := mock.Called(targetDir, processModuleConfig)
	return args.Error(0)
}
