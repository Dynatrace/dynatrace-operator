package server

import (
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	metricsBindAddress   = ":8080"
	defaultProbeAddress  = ":10080"
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
	options := ctrl.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
		Scheme:                 scheme.Scheme,
		HealthProbeBindAddress: provider.probeAddress,
		LivenessEndpointName:   livenessEndpointName,
	}

	return options
}
