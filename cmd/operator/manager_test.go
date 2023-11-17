package operator

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	mockedmanager "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/manager"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

const (
	addHealthzCheckMethodName = "AddHealthzCheck"
	checkerArgumentType       = "healthz.Checker"
)

func TestOperatorManagerProvider(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		var controlManagerProvider manager.Provider = NewOperatorManagerProvider(false)
		_, _ = controlManagerProvider.CreateManager("namespace", &rest.Config{})
	})

	t.Run("creates correct options", func(t *testing.T) {
		operatorMgrProvider := operatorManagerProvider{}
		options := operatorMgrProvider.createOptions("namespace")

		assert.NotNil(t, options)

		assert.Contains(t, options.Cache.DefaultNamespaces, "namespace")
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, metricsBindAddress, options.Metrics.BindAddress)

		assert.True(t, options.LeaderElection)
		assert.Equal(t, leaderElectionId, options.LeaderElectionID)
		assert.Equal(t, leaderElectionResourceLock, options.LeaderElectionResourceLock)
		assert.Equal(t, "namespace", options.LeaderElectionNamespace)
		assert.Equal(t, healthProbeBindAddress, options.HealthProbeBindAddress)
		assert.Equal(t, livenessEndpointName, options.LivenessEndpointName)
	})
	t.Run("check if healthz/readyz checks are added", func(t *testing.T) {
		testHealthzAndReadyz(t, func(mockMgr *mockedmanager.Manager) error {
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
		testBootstrapHealthzAndReadyz(t, func(mockMgr *mockedmanager.Manager) error {
			bootstrapProvider, _ := NewBootstrapManagerProvider().(bootstrapManagerProvider)
			bootstrapProvider.setManager(mockMgr)
			_, err := bootstrapProvider.CreateManager("namespace", &rest.Config{})
			return err
		})
	})
}

func testHealthzAndReadyz(t *testing.T, createProviderAndRunManager func(mockMgr *mockedmanager.Manager) error) {
	mockMgr := mockedmanager.NewManager(t)
	mockMgr.On(addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType)).Return(nil)

	client := fake.NewClient(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}})
	mockMgr.On("GetConfig").Return(&rest.Config{})
	mockMgr.On("GetScheme").Return(scheme.Scheme)
	mockMgr.On("GetClient").Return(client)
	mockMgr.On("GetAPIReader").Return(client)
	mockMgr.On("GetControllerOptions").Return(config.Controller{})
	mockMgr.On("GetLogger").Return(logr.Logger{})
	mockMgr.On("Add", mock.AnythingOfType("*controller.Controller")).Return(nil)
	mockMgr.On("GetCache").Return(nil)
	mockMgr.On("GetRESTMapper").Return(nil)

	err := createProviderAndRunManager(mockMgr)

	assert.NoError(t, err)
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType))

	expectedHealthzError := errors.New("healthz error")
	mockMgr = mockedmanager.NewManager(t)
	mockMgr.On(addHealthzCheckMethodName, mock.Anything, mock.Anything).Return(expectedHealthzError)

	err = createProviderAndRunManager(mockMgr)

	assert.EqualError(t, err, expectedHealthzError.Error())
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, mock.Anything, mock.Anything)
}

func testBootstrapHealthzAndReadyz(t *testing.T, createProviderAndRunManager func(mockMgr *mockedmanager.Manager) error) {
	mockMgr := mockedmanager.NewManager(t)
	mockMgr.On(addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType)).Return(nil)

	err := createProviderAndRunManager(mockMgr)

	assert.NoError(t, err)
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, livezEndpointName, mock.AnythingOfType(checkerArgumentType))

	expectedHealthzError := errors.New("healthz error")
	mockMgr = mockedmanager.NewManager(t)
	mockMgr.On(addHealthzCheckMethodName, mock.Anything, mock.Anything).Return(expectedHealthzError)

	err = createProviderAndRunManager(mockMgr)

	assert.EqualError(t, err, expectedHealthzError.Error())
	mockMgr.AssertCalled(t, addHealthzCheckMethodName, mock.Anything, mock.Anything)
}
