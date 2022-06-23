package operator

import (
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	metricsBindAddress     = ":8080"
	healthProbeBindAddress = ":10080"
	operatorManagerPort    = 8383

	leaderElectionId           = "dynatrace-operator-lock"
	leaderElectionResourceLock = "configmaps"

	livenessEndpointName = "/livez"
)

type managerProvider interface {
	CreateManager(namespace string, config *rest.Config) (manager.Manager, error)
}

type bootstrapManagerProvider struct{}

func (provider bootstrapManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme.Scheme,
		Namespace: namespace,
	})
	return controlManager, errors.WithStack(err)
}

func newBootstrapManagerProvider() managerProvider {
	return bootstrapManagerProvider{}
}

type operatorManagerProvider struct{}

func (provider operatorManagerProvider) CreateManager(namespace string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:                  namespace,
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         metricsBindAddress,
		Port:                       operatorManagerPort,
		LeaderElection:             true,
		LeaderElectionID:           leaderElectionId,
		LeaderElectionResourceLock: leaderElectionResourceLock,
		LeaderElectionNamespace:    namespace,
		HealthProbeBindAddress:     healthProbeBindAddress,
		LivenessEndpointName:       livenessEndpointName,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = mgr.AddHealthzCheck("livez", healthz.Ping); err != nil {
		return nil, errors.WithStack(err)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, errors.WithStack(err)
	}

	return mgr, nil
}

func newOperatorManagerProvider() managerProvider {
	return operatorManagerProvider{}
}
