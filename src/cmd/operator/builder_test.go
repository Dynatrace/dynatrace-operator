package operator

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "test-namespace"
	testPod       = "test-pod-name"
)

func TestCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewOperatorCommandBuilder()
		operatorCommand := builder.Build()

		assert.NotNil(t, operatorCommand)
		assert.Equal(t, use, operatorCommand.Use)
		assert.NotNil(t, operatorCommand.RunE)
	})
	t.Run("set config provider", func(t *testing.T) {
		builder := NewOperatorCommandBuilder()

		assert.NotNil(t, builder)

		expectedProvider := &config.MockProvider{}
		builder = builder.SetConfigProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.configProvider)
	})
	t.Run("set operator manager provider", func(t *testing.T) {
		expectedProvider := &cmdManager.MockProvider{}
		builder := NewOperatorCommandBuilder().setOperatorManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.operatorManagerProvider)
	})
	t.Run("set bootstrap manager provider", func(t *testing.T) {
		expectedProvider := &cmdManager.MockProvider{}
		builder := NewOperatorCommandBuilder().setBootstrapManagerProvider(expectedProvider)

		assert.Equal(t, expectedProvider, builder.bootstrapManagerProvider)
	})
	t.Run("set namespace", func(t *testing.T) {
		builder := NewOperatorCommandBuilder().SetNamespace("namespace")

		assert.Equal(t, "namespace", builder.namespace)
	})
	t.Run("set context", func(t *testing.T) {
		// If ctrl.SetupSignalHandler() is used multiple times during a test suit, it will panic
		// Therefore it is necessary to set a custom context to unit test properly
		ctx := context.TODO()
		builder := NewOperatorCommandBuilder().setSignalHandler(ctx)

		assert.Equal(t, ctx, builder.signalHandler)
	})
}

func TestOperatorCommand(t *testing.T) {
	t.Run("operator command exists", func(t *testing.T) {
		operatorCommand := NewOperatorCommandBuilder().Build()

		assert.Equal(t, operatorCommand.Use, "operator")
		assert.NotNil(t, operatorCommand.RunE)
	})
	t.Run("kubernetes config provider is called", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &cmdManager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&cmdManager.TestManager{}, nil)

		builder := NewOperatorCommandBuilder().
			SetNamespace(testNamespace).
			setOperatorManagerProvider(mockMgrProvider).
			setBootstrapManagerProvider(mockMgrProvider).
			SetConfigProvider(mockCfgProvider)
		operatorCommand := builder.Build()

		_ = operatorCommand.RunE(operatorCommand, make([]string, 0))

		mockCfgProvider.AssertCalled(t, "GetConfig")
	})
	t.Run("exit on config provider error", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, errors.New("config provider error"))
		builder := NewOperatorCommandBuilder().
			SetConfigProvider(mockCfgProvider)
		operatorCommand := builder.Build()

		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.EqualError(t, err, "config provider error")
	})
	t.Run("create manager if not in OLM", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &cmdManager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&cmdManager.TestManager{}, nil)

		builder := NewOperatorCommandBuilder().
			SetNamespace(testNamespace).
			SetPodName(testPod).
			setOperatorManagerProvider(mockMgrProvider).
			setBootstrapManagerProvider(mockMgrProvider).
			SetConfigProvider(mockCfgProvider).
			setSignalHandler(context.TODO()).
			setClient(createFakeClient(false))
		operatorCommand := builder.Build()

		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
	})
	t.Run("exit on manager error", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgrProvider := &cmdManager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(&cmdManager.TestManager{}, errors.New("create manager error"))

		builder := NewOperatorCommandBuilder().
			SetNamespace(testNamespace).
			SetPodName(testPod).
			setBootstrapManagerProvider(mockMgrProvider).
			SetConfigProvider(mockCfgProvider).
			setClient(createFakeClient(false))
		operatorCommand := builder.Build()

		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.EqualError(t, err, "create manager error")
	})
	t.Run("bootstrap manager is started", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		mockMgr := &cmdManager.MockManager{}
		mockMgr.On("Start", mock.Anything).Return(nil)

		mockMgrProvider := &cmdManager.MockProvider{}
		mockMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(mockMgr, nil)

		builder := NewOperatorCommandBuilder().
			SetNamespace(testNamespace).
			SetPodName(testPod).
			setOperatorManagerProvider(mockMgrProvider).
			setBootstrapManagerProvider(mockMgrProvider).
			SetConfigProvider(mockCfgProvider).
			setSignalHandler(context.TODO()).
			setClient(createFakeClient(false))
		operatorCommand := builder.Build()

		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, "Start", mock.Anything)
	})
	t.Run("operator manager is started", func(t *testing.T) {
		mockCfgProvider := &config.MockProvider{}
		mockCfgProvider.On("GetConfig").Return(&rest.Config{}, nil)

		bootstrapMockMgr := &cmdManager.MockManager{}
		bootstrapMockMgr.On("Start", mock.Anything).Return(nil)

		mockBootstrapMgrProvider := &cmdManager.MockProvider{}
		mockBootstrapMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(bootstrapMockMgr, nil)

		operatorMockMgr := &cmdManager.MockManager{}
		operatorMockMgr.On("Start", mock.Anything).Return(nil)

		mockOperatorMgrProvider := &cmdManager.MockProvider{}
		mockOperatorMgrProvider.
			On("CreateManager", mock.AnythingOfType("string"), &rest.Config{}).
			Return(operatorMockMgr, nil)

		builder := NewOperatorCommandBuilder().
			SetNamespace(testNamespace).
			SetPodName(testPod).
			setOperatorManagerProvider(mockOperatorMgrProvider).
			setBootstrapManagerProvider(mockBootstrapMgrProvider).
			SetConfigProvider(mockCfgProvider).
			setSignalHandler(context.TODO()).
			setClient(createFakeClient(true))
		operatorCommand := builder.Build()

		err := operatorCommand.RunE(operatorCommand, make([]string, 0))

		assert.NoError(t, err)
		bootstrapMockMgr.AssertNotCalled(t, "Start", mock.Anything)
		operatorMockMgr.AssertCalled(t, "Start", mock.Anything)
	})
}

func createFakeClient(isDeployedViaOlm bool) client.WithWatch {
	var annotations map[string]string = map[string]string{}
	if isDeployedViaOlm {
		annotations = map[string]string{
			"olm.operatorNamespace": "operators",
		}
	}

	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			},
			&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        testPod,
					Namespace:   testNamespace,
					Annotations: annotations,
				},
			},
		).Build()
}
