package operator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

const (
	testNamespace = "test-namespace"
)

type mockManager struct {
	testManager
	mock.Mock
}

func (mgr *mockManager) Start(ctx context.Context) error {
	args := mgr.Called(ctx)
	return args.Error(0)
}

func TestOperatorCommand(t *testing.T) {
	t.Run("operator command exists", func(t *testing.T) {
		operatorCommand := newOperatorCommand(runConfig{})

		assert.Equal(t, operatorCommand.Use, "operator")
		assert.NotNil(t, operatorCommand.RunE)
	})
	t.Run("kubernetes config provider is called", func(t *testing.T) {
		mockCfgProvider := &mockConfigProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &mockManagerProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&testManager{}, nil)

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
		mockCfgProvider := &mockConfigProvider{}
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
		mockCfgProvider := &mockConfigProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &mockManagerProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&testManager{}, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			operatorManagerProvider:  mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
			signalHandler:            context.TODO(),
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
	})
	t.Run("exit on manager error", func(t *testing.T) {
		mockCfgProvider := &mockConfigProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &mockManagerProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&testManager{}, errors.New("create manager error"))

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
		mockCfgProvider := &mockConfigProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgr := &mockManager{}
		mockMgr.On("Start", mock.Anything).Return(nil)

		mockMgrProvider := &mockManagerProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(mockMgr, nil)

		runCfg := runConfig{
			kubeConfigProvider:       mockCfgProvider,
			bootstrapManagerProvider: mockMgrProvider,
			operatorManagerProvider:  mockMgrProvider,
			isDeployedInOlm:          false,
			namespace:                testNamespace,
			signalHandler:            context.TODO(),
		}
		operatorCommand := newOperatorCommand(runCfg)
		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, "Start", mock.Anything)
	})
	t.Run("operator manager is started", func(t *testing.T) {
		mockCfgProvider := &mockConfigProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		bootstrapMockMgr := &mockManager{}
		bootstrapMockMgr.On("Start", mock.Anything).Return(nil)

		mockBootstrapMgrProvider := &mockManagerProvider{}
		mockBootstrapMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(bootstrapMockMgr, nil)

		operatorMockMgr := &mockManager{}
		operatorMockMgr.On("Start", mock.Anything).Return(nil)

		mockOperatorMgrProvider := &mockManagerProvider{}
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
