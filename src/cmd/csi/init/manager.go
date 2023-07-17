package init

import (

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func createManager(namespace string, config *rest.Config) (manager.Manager, error) {
	mgr, err := manager.New(config, createManagerOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func createManagerOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Namespace: namespace,
		Scheme:    scheme.Scheme,
	}
}
