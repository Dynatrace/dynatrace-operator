package operator

import (
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type managerProvider interface {
	CreateManager(namespace string, config *rest.Config) (manager.Manager, error)
}

type bootstrapManagerProvider struct {
}

func (provider bootstrapManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme.Scheme,
		Namespace: namespace,
	})
	return controlManager, errors.WithStack(err)
}

func newControlManagerProvider() managerProvider {
	return bootstrapManagerProvider{}
}
