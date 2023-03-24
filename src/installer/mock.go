package installer

import (
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

var _ Installer = &Mock{}

func (mock *Mock) InstallAgent(targetDir string) (bool, error) {
	args := mock.Called(targetDir)
	return args.Bool(0), args.Error(1)
}

func (mock *Mock) UpdateProcessModuleConfig(targetDir string, processModuleConfig *dtclient.ProcessModuleConfig) error {
	args := mock.Called(targetDir, processModuleConfig)
	return args.Error(0)
}

func (mock *Mock) Cleanup() error {
	args := mock.Called()
	return args.Error(0)
}
