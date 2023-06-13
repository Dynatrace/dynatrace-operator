package webhook

import (
	"crypto/tls"

	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	metricsBindAddress     = ":8383"
	healthProbeBindAddress = ":10080"
	port                   = 8443
	livezEndpointName      = "livez"
	livenessEndpointName   = "/" + livezEndpointName
	readyzEndpointName     = "readyz"
	readinessEndpointName  = "/" + readyzEndpointName
)

type Provider struct {
	certificateDirectory string
	certificateFileName  string
	keyFileName          string
}

func NewProvider(certificateDirectory string, keyFileName string, certificateFileName string) Provider {
	return Provider{
		certificateDirectory: certificateDirectory,
		certificateFileName:  certificateFileName,
		keyFileName:          keyFileName,
	}
}

func (provider Provider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(config, provider.createOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = controlManager.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = controlManager.AddReadyzCheck(readyzEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return provider.setupWebhookServer(controlManager), nil
}

func (provider Provider) createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Scheme:                 scheme.Scheme,
		Namespace:              namespace,
		MetricsBindAddress:     metricsBindAddress,
		ReadinessEndpointName:  readinessEndpointName,
		LivenessEndpointName:   livenessEndpointName,
		HealthProbeBindAddress: healthProbeBindAddress,
		Port:                   port,
	}
}

func (provider Provider) setupWebhookServer(mgr manager.Manager) manager.Manager {
	tlsConfig := func(config *tls.Config) {
		config.MinVersion = tls.VersionTLS13
	}

	webhookServer := mgr.GetWebhookServer()
	webhookServer.CertDir = provider.certificateDirectory
	webhookServer.KeyName = provider.keyFileName
	webhookServer.CertName = provider.certificateFileName
	webhookServer.TLSOpts = []func(*tls.Config){tlsConfig}

	return mgr
}
