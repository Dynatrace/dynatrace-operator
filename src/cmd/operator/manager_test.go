package operator

import (
	"errors"
	"testing"

	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	t.Run("check if healthz/readyz checks are added", func(t *testing.T) {
		testHealthzAndReadyz(t, func(mockMgr *cmdManager.MockManager) error {
			var controlManagerProvider = NewOperatorManagerProvider(false).(operatorManagerProvider)
			controlManagerProvider.setManager(mockMgr)
			_, err := controlManagerProvider.CreateManager("namespace", &rest.Config{})
			return err
		})
	})
}

func TestBootstrapManagerProvider(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		bootstrapProvider := NewBootstrapManagerProvider()
		_, _ = bootstrapProvider.CreateManager("namespace", &rest.Config{})
	})
	t.Run("check if healthz/readyz checks are added", func(t *testing.T) {
		testHealthzAndReadyz(t, func(mockMgr *cmdManager.MockManager) error {
			bootstrapProvider := NewBootstrapManagerProvider().(bootstrapManagerProvider)
			bootstrapProvider.setManager(mockMgr)
			_, err := bootstrapProvider.CreateManager("namespace", &rest.Config{})
			return err
		})
	})
}

func testHealthzAndReadyz(t *testing.T, createProviderAndRunManager func(mockMgr *cmdManager.MockManager) error) {
	const addHealthzCheckMethodName = "AddHealthzCheck"
	const addReadyzCheckMethodName = "AddReadyzCheck"
	const checkerArgumentType = "healthz.Checker"

	mockMgr := &cmdManager.MockManager{}
	mockMgr.On(addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType)).Return(nil)
	mockMgr.On(addReadyzCheckMethodName, readyzEndpointName, mock.AnythingOfType(checkerArgumentType)).Return(nil)

	client := fake.NewClient(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}})
	mockMgr.On("GetConfig").Return(&rest.Config{})
	mockMgr.On("GetScheme").Return(scheme.Scheme)
	mockMgr.On("GetClient").Return(client)
	mockMgr.On("GetAPIReader").Return(client)

	err := createProviderAndRunManager(mockMgr)

	assert.NoError(t, err)
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType))
	mockMgr.AssertCalled(t, addReadyzCheckMethodName, readyzEndpointName, mock.AnythingOfType(checkerArgumentType))

	expectedHealthzError := errors.New("healthz error")
	mockMgr = &cmdManager.MockManager{}
	mockMgr.On(addHealthzCheckMethodName, mock.Anything, mock.Anything).Return(expectedHealthzError)

	err = createProviderAndRunManager(mockMgr)

	assert.EqualError(t, err, expectedHealthzError.Error())
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, mock.Anything, mock.Anything)
}
