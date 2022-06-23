package operator

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
)

type testManager struct {
	manager.Manager
}

func (mgr *testManager) GetClient() client.Client {
	return struct{ client.Client }{}
}

func (mgr *testManager) GetAPIReader() client.Reader {
	return struct{ client.Reader }{}
}

func (mgr *testManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return v1alpha1.ControllerConfigurationSpec{}
}

func (mgr *testManager) GetScheme() *runtime.Scheme {
	return scheme.Scheme
}

func (mgr *testManager) GetLogger() logr.Logger {
	return logger.NewDTLogger()
}

func (mgr *testManager) SetFields(interface{}) error {
	return nil
}

func (mgr *testManager) Add(manager.Runnable) error {
	return nil
}

func (mgr *testManager) Start(_ context.Context) error {
	return nil
}

type mockManagerProvider struct {
	mock.Mock
}

func (provider *mockManagerProvider) CreateManager(namespace string, cfg *rest.Config) (manager.Manager, error) {
	args := provider.Called(namespace, cfg)
	return args.Get(0).(manager.Manager), args.Error(1)
}

func TestManagerProvider(t *testing.T) {
	bootstrapProvider := newBootstrapManagerProvider()
	_, _ = bootstrapProvider.CreateManager("namespace", &rest.Config{})

	controlManagerProvider := newOperatorManagerProvider()
	_, _ = controlManagerProvider.CreateManager("namespace", &rest.Config{})
}
