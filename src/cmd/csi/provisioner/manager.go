package provisioner

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
	metricsBindAddress   = ":8090"
	defaultProbeAddress  = ":8091"
	livenessEndpointName = "/livez"
	livezEndpointName    = "livez"
)

type csiDriverManagerProvider struct {
	probeAddress string
}

func newCsiDriverManagerProvider(probeAddress string) cmdManager.Provider {
	return csiDriverManagerProvider{
		probeAddress: probeAddress,
	}
}

func (provider csiDriverManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	mgr, err := manager.New(config, provider.createOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = provider.addHealthzCheck(mgr)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (provider csiDriverManagerProvider) addHealthzCheck(mgr manager.Manager) error {
	err := mgr.AddHealthzCheck(livezEndpointName, healthz.Ping)

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (provider csiDriverManagerProvider) createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Namespace:              namespace,
		Scheme:                 scheme.Scheme,
		MetricsBindAddress:     metricsBindAddress,
		HealthProbeBindAddress: provider.probeAddress,
		LivenessEndpointName:   livenessEndpointName,
	}
}
