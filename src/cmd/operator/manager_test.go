package operator

import (
	"testing"

	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/rest"
)

func TestOperatorManagerProvider(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		var controlManagerProvider cmdManager.Provider = NewOperatorManagerProvider(false)
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
		const addHealthzCheckMethodName = "AddHealthzCheck"

		mockMgr := &cmdManager.MockManager{}
		mockMgr.On(addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType("healthz.Checker")).Return(nil)

		err := addHealthzCheck(mockMgr)

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType("healthz.Checker"))

		expectedError := errors.New("healthz error")
		mockMgr = &cmdManager.MockManager{}
		mockMgr.On(addHealthzCheckMethodName, mock.Anything, mock.Anything).Return(expectedError)

		err = addHealthzCheck(mockMgr)

		assert.EqualError(t, err, expectedError.Error())
		mockMgr.AssertCalled(t, addHealthzCheckMethodName, mock.Anything, mock.Anything)
	})
	t.Run("adds readyz check endpoint", func(t *testing.T) {
		const addReadyzCheckMethodName = "AddReadyzCheck"

		mockMgr := &cmdManager.MockManager{}
		mockMgr.On(addReadyzCheckMethodName, readyzEndpointName, mock.AnythingOfType("healthz.Checker")).Return(nil)

		err := addReadyzCheck(mockMgr)

		assert.NoError(t, err)
		mockMgr.AssertCalled(t, addReadyzCheckMethodName, readyzEndpointName, mock.AnythingOfType("healthz.Checker"))

		expectedError := errors.New("readyz error")
		mockMgr = &cmdManager.MockManager{}
		mockMgr.On(addReadyzCheckMethodName, mock.Anything, mock.Anything).Return(expectedError)

		err = addReadyzCheck(mockMgr)

		assert.EqualError(t, err, expectedError.Error())
		mockMgr.AssertCalled(t, addReadyzCheckMethodName, mock.Anything, mock.Anything)
	})
}

func TestBootstrapManagerProvider(t *testing.T) {
	bootstrapProvider := NewBootstrapManagerProvider()
	_, _ = bootstrapProvider.CreateManager("namespace", &rest.Config{})

}
