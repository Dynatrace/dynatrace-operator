package provisioner

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	managermock "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/manager"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCsiDriverManagerProvider(t *testing.T) {
	t.Run("is instantiable", func(t *testing.T) {
		csiManagerProvider := newCsiDriverManagerProvider(defaultProbeAddress)
		assert.NotNil(t, csiManagerProvider)

		csiManagerProviderImpl := csiManagerProvider.(csiDriverManagerProvider)
		assert.Equal(t, defaultProbeAddress, csiManagerProviderImpl.probeAddress)
	})
	t.Run("creates options", func(t *testing.T) {
		csiManagerProvider := csiDriverManagerProvider{}

		options := csiManagerProvider.createOptions("namespace")

		assert.NotNil(t, options)
		assert.Contains(t, options.Cache.DefaultNamespaces, "namespace")
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, metricsBindAddress, options.Metrics.BindAddress)
		assert.Equal(t, "", options.HealthProbeBindAddress)
		assert.Equal(t, livenessEndpointName, options.LivenessEndpointName)
	})
	t.Run("adds healthz check endpoint", func(t *testing.T) {
		const addHealthzCheck = "AddHealthzCheck"

		operatorMgrProvider := csiDriverManagerProvider{}
		mockMgr := managermock.NewManager(t)
		mockMgr.On(addHealthzCheck, livezEndpointName, mock.AnythingOfType("healthz.Checker")).Return(nil)

		err := operatorMgrProvider.addHealthzCheck(mockMgr)

		require.NoError(t, err)
		mockMgr.AssertCalled(t, addHealthzCheck, livezEndpointName, mock.AnythingOfType("healthz.Checker"))

		expectedError := errors.New("healthz error")
		mockMgr = managermock.NewManager(t)
		mockMgr.On(addHealthzCheck, mock.Anything, mock.Anything).Return(expectedError)

		err = operatorMgrProvider.addHealthzCheck(mockMgr)

		require.EqualError(t, err, expectedError.Error())
		mockMgr.AssertCalled(t, addHealthzCheck, mock.Anything, mock.Anything)
	})
}
