package operator

import (
	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
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
	readyzEndpointName   = "readyz"
	livezEndpointName    = "livez"
)

type bootstrapManagerProvider struct{}

func (provider bootstrapManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:    scheme.Scheme,
		Namespace: namespace,
	})
	return controlManager, errors.WithStack(err)
}

func newBootstrapManagerProvider() cmdManager.Provider {
	return bootstrapManagerProvider{}
}

type operatorManagerProvider struct{}

func (provider operatorManagerProvider) CreateManager(namespace string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, provider.createOptions(namespace))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = provider.addHealthzCheck(mgr)
	if err != nil {
		return nil, err
	}

	err = provider.addReadyzCheck(mgr)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (provider operatorManagerProvider) createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
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
	}
}

func (provider operatorManagerProvider) addHealthzCheck(mgr manager.Manager) error {
	err := mgr.AddHealthzCheck(livezEndpointName, healthz.Ping)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (provider operatorManagerProvider) addReadyzCheck(mgr manager.Manager) error {
	err := mgr.AddReadyzCheck(readyzEndpointName, healthz.Ping)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func newOperatorManagerProvider() cmdManager.Provider {
	return operatorManagerProvider{}
}
