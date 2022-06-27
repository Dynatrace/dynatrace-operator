package operator

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

const (
	testNamespace = "test-namespace"
)

func TestOperatorCommand(t *testing.T) {
	t.Run("operator command exists", func(t *testing.T) {
		operatorCommand := newOperatorCommand(runConfig{})

		assert.Equal(t, operatorCommand.Use, "operator")
		assert.NotNil(t, operatorCommand.RunE)
	})
	t.Run("kubernetes config provider is called", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &manager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&manager.TestManager{}, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			operatorManagerProvider:  mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
		}
		operatorCommand := newOperatorCommand(runCfg)

		_ = operatorCommand.RunE(operatorCommand, make([]string, 0))

		mockCfgProvider.AssertCalled(t, "GetConfig")
	})
	t.Run("exit on config provider error", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, errors.New("config provider error"))
		runCfg := runConfig{
			kubeConfigProvider: mockCfgProvider,
			isDeployedInOlm:    false,
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.EqualError(t, err, "config provider error")
	})
	t.Run("create manager if not in OLM", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &manager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&manager.TestManager{}, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			operatorManagerProvider:  mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
	})
	t.Run("exit on manager error", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &manager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&manager.TestManager{}, errors.New("create manager error"))

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.EqualError(t, err, "create manager error")
	})
	t.Run("bootstrap manager is started", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgr := &manager.MockManager{}
		mockMgr.On("Start", mock.Anything).Return(nil)

		mockMgrProvider := &manager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(mockMgr, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			operatorManagerProvider:  mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, "Start", mock.Anything)
	})
	t.Run("operator manager is started", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		bootstrapMockMgr := &manager.MockManager{}
		bootstrapMockMgr.On("Start", mock.Anything).Return(nil)

		mockBootstrapMgrProvider := &manager.MockProvider{}
		mockBootstrapMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(bootstrapMockMgr, nil)

		operatorMockMgr := &manager.MockManager{}
		operatorMockMgr.On("Start", mock.Anything).Return(nil)

		mockOperatorMgrProvider := &manager.MockProvider{}
		mockOperatorMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(operatorMockMgr, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockBootstrapMgrProvider,
			operatorManagerProvider:  mockOperatorMgrProvider,
			isDeployedInOlm:          true,
			namespace:                testNamespace,
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
		bootstrapMockMgr.AssertNotCalled(t, "Start", mock.Anything)
		operatorMockMgr.AssertCalled(t, "Start", mock.Anything)
	})
}
