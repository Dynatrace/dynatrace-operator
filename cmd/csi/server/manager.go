package server

import (
	cmdManager "github.com/Dynatrace/dynatrace-operator/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	metricsBindAddress = ":8080"
)

type csiDriverManagerProvider struct{}

func newCsiDriverManagerProvider() cmdManager.Provider {
	return csiDriverManagerProvider{}
}

func (provider csiDriverManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	mgr, err := manager.New(config, provider.createOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// instrument webhook manager HTTP client with OpenTelemetry
	mgr.GetHTTPClient().Transport = otelhttp.NewTransport(mgr.GetHTTPClient().Transport)

	return mgr, nil
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
		Scheme: scheme.Scheme,
	}

	return options
}
