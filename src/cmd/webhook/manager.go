package webhook

import (
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	metricsBindAddress = ":8383"
	port               = 8443
)

type webhookProvider struct {
	certificateDirectory string
	certificateFileName  string
	keyFileName          string
}

func newWebhookProvider(certificateDirectory string, keyFileName string, certificateFileName string) webhookProvider {
	return webhookProvider{
		certificateDirectory: certificateDirectory,
		certificateFileName:  certificateFileName,
		keyFileName:          keyFileName,
	}
}

func (provider webhookProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(config, provider.createOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return provider.setupWebhookServer(mgr), nil
}

func (provider webhookProvider) createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Namespace:          namespace,
		Scheme:             scheme.Scheme,
		MetricsBindAddress: metricsBindAddress,
		Port:               port,
	}
}

func (provider webhookProvider) setupWebhookServer(mgr manager.Manager) manager.Manager {
	webhookServer := mgr.GetWebhookServer()
	webhookServer.CertDir = provider.certificateDirectory
	webhookServer.KeyName = provider.keyFileName
	webhookServer.CertName = provider.certificateFileName

	return mgr
}
