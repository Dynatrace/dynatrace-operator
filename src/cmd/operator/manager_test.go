package operator

import (
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type mockManagerProvider struct {
	mock.Mock
}

func (provider *mockManagerProvider) CreateManager(namespace string, cfg *rest.Config) (manager.Manager, error) {
	args := provider.Called(namespace, cfg)
	return args.Get(0).(manager.Manager), args.Error(1)
}

func TestOperatorManagerProvider(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		var controlManagerProvider cmdManager.Provider = newOperatorManagerProvider()
		_, _ = controlManagerProvider.CreateManager("namespace", &rest.Config{})
	})
	t.Run("creates correct options", func(t *testing.T) {
		operatorMgrProvider := operatorManagerProvider{}
		options := operatorMgrProvider.createOptions("namespace")

		assert.NotNil(t, options)
		assert.Equal(t, "namespace", options.Namespace)
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, metricsBindAddress, options.MetricsBindAddress)
		assert.Equal(t, operatorManagerPort, options.Port)
		assert.True(t, options.LeaderElection)
		assert.Equal(t, leaderElectionId, options.LeaderElectionID)
		assert.Equal(t, leaderElectionResourceLock, options.LeaderElectionResourceLock)
		assert.Equal(t, "namespace", options.LeaderElectionNamespace)
		assert.Equal(t, healthProbeBindAddress, options.HealthProbeBindAddress)
		assert.Equal(t, livenessEndpointName, options.LivenessEndpointName)
	})
	t.Run("adds healthz check endpoint", func(t *testing.T) {
		const addHealthzCheck = "AddHealthzCheck"

		operatorMgrProvider := operatorManagerProvider{}
		mockMgr := &cmdManager.Mock{}
		mockMgr.On(addHealthzCheck, livezEndpointName, mock.AnythingOfType("healthz.Checker")).Return(nil)

		err := operatorMgrProvider.addHealthzCheck(mockMgr)

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, addHealthzCheck, livezEndpointName, mock.AnythingOfType("healthz.Checker"))

		expectedError := errors.New("healthz error")
		mockMgr = &cmdManager.Mock{}
		mockMgr.On(addHealthzCheck, mock.Anything, mock.Anything).Return(expectedError)

		err = operatorMgrProvider.addHealthzCheck(mockMgr)

		assert.EqualError(t, err, expectedError.Error())
		mockMgr.AssertCalled(t, addHealthzCheck, mock.Anything, mock.Anything)
	})
	t.Run("adds readyz check endpoint", func(t *testing.T) {
		const addReadyzCheck = "AddReadyzCheck"

		operatorMgrProvider := operatorManagerProvider{}
		mockMgr := &cmdManager.Mock{}
		mockMgr.On(addReadyzCheck, readyzEndpointName, mock.AnythingOfType("healthz.Checker")).Return(nil)

		err := operatorMgrProvider.addReadyzCheck(mockMgr)

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, addReadyzCheck, readyzEndpointName, mock.AnythingOfType("healthz.Checker"))

		expectedError := errors.New("readyz error")
		mockMgr = &cmdManager.Mock{}
		mockMgr.On(addReadyzCheck, mock.Anything, mock.Anything).Return(expectedError)

		err = operatorMgrProvider.addReadyzCheck(mockMgr)

		assert.EqualError(t, err, expectedError.Error())
		mockMgr.AssertCalled(t, addReadyzCheck, mock.Anything, mock.Anything)
	})
}

func TestBootstrapManagerProvider(t *testing.T) {
	bootstrapProvider := newBootstrapManagerProvider()
	_, _ = bootstrapProvider.CreateManager("namespace", &rest.Config{})

}
