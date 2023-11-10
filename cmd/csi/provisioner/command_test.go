package provisioner

import (
	"testing"

	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	dtfake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/Dynatrace/dynatrace-operator/test/mocks/cmd/config"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

func TestCsiCommand(t *testing.T) {
	configProvider := mocks.NewProvider(t)
	configProvider.On("GetConfig").Return(&rest.Config{}, nil)

	clt := dtfake.NewClient()
	cmdMgr := &cmdManager.MockManager{}
	cmdMgr.On("GetAPIReader", mock.Anything, mock.Anything).Return(clt, nil)

	managerProvider := &cmdManager.MockProvider{}
	managerProvider.On("CreateManager", mock.Anything, mock.Anything).Return(cmdMgr, nil)

	memFs := afero.NewMemMapFs()
	builder := NewCsiProvisionerCommandBuilder().
		SetConfigProvider(configProvider).
		setManagerProvider(managerProvider).
		SetNamespace("test-namespace").
		setFilesystem(memFs)
	command := builder.Build()
	commandFn := builder.buildRun()

	err := commandFn(command, make([]string, 0))

	// sqlite library does not use afero fs, so it throws an error because path does not exist
	assert.Error(t, err)
	configProvider.AssertCalled(t, "GetConfig")
	managerProvider.AssertCalled(t, "CreateManager", "test-namespace", &rest.Config{})

	exists, err := afero.Exists(memFs, dtcsi.DataPath)
	assert.True(t, exists)
	assert.NoError(t, err)

	// Logging a newline because otherwise `go test` doesn't recognize the result
	logger.Factory.GetLogger("csi command").Info("")
}
