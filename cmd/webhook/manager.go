package webhook

import (
	"crypto/tls"
	"os"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	defaultMetricsBindAddress     = ":8383"
	defaultHealthProbeBindAddress = ":10080"
	defaultPort                   = 8443
	livezEndpointName             = "livez"
	livenessEndpointName          = "/" + livezEndpointName
	readyzEndpointName            = "readyz"
	readinessEndpointName         = "/" + readyzEndpointName
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

	return provider.setupWebhookServer(controlManager)
}

func (provider Provider) createOptions(namespace string) ctrl.Options {
	port := defaultPort
	webhookPortEnv := os.Getenv("WEBHOOK_PORT")

	if parsedWebhookPort, err := strconv.Atoi(webhookPortEnv); err == nil {
		port = parsedWebhookPort
	}

	metricsBindAddress := defaultMetricsBindAddress

	metricsBindAddressEnv := os.Getenv("METRICS_BIND_ADDRESS")
	if metricsBindAddressEnv != "" {
		metricsBindAddress = metricsBindAddressEnv
	}

	healthProbeBindAddress := defaultHealthProbeBindAddress

	healthProbeBindAddressEnv := os.Getenv("HEALTH_PROBE_BIND_ADDRESS")
	if healthProbeBindAddressEnv != "" {
		healthProbeBindAddress = healthProbeBindAddressEnv
	}

	return ctrl.Options{
		Scheme:                 scheme.Scheme,
		ReadinessEndpointName:  readinessEndpointName,
		LivenessEndpointName:   livenessEndpointName,
		HealthProbeBindAddress: healthProbeBindAddress,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: port,
		}),
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
	}
}

func (provider Provider) setupWebhookServer(mgr manager.Manager) (manager.Manager, error) {
	tlsConfig := func(config *tls.Config) {
		config.MinVersion = tls.VersionTLS13
	}

	webhookServer, ok := mgr.GetWebhookServer().(*webhook.DefaultServer)
	if !ok {
		return nil, errors.WithStack(errors.New("Unable to cast webhook server"))
	}

	webhookServer.Options.CertDir = provider.certificateDirectory
	webhookServer.Options.KeyName = provider.keyFileName
	webhookServer.Options.CertName = provider.certificateFileName
	webhookServer.Options.TLSOpts = []func(*tls.Config){tlsConfig}

	return mgr, nil
}
