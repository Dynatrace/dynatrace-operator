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

func createManager(config *rest.Config, namespace, certificateDirectory, certificateFileName, keyFileName string) (manager.Manager, error) {
	controlManager, err := ctrl.NewManager(config, createOptions(namespace))
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

	return setupWebhookServer(controlManager, certificateDirectory, certificateFileName, keyFileName)
}

func createOptions(namespace string) ctrl.Options {
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
		PprofBindAddress: os.Getenv("PPROF_BIND_ADDRESS"),
	}
}

func setupWebhookServer(mgr manager.Manager, certificateDirectory, certificateFileName, keyFileName string) (manager.Manager, error) {
	tlsConfig := func(config *tls.Config) {
		config.MinVersion = tls.VersionTLS13
	}

	webhookServer, ok := mgr.GetWebhookServer().(*webhook.DefaultServer)
	if !ok {
		return nil, errors.WithStack(errors.New("Unable to cast webhook server"))
	}

	webhookServer.Options.CertDir = certificateDirectory
	webhookServer.Options.KeyName = keyFileName
	webhookServer.Options.CertName = certificateFileName
	webhookServer.Options.TLSOpts = []func(*tls.Config){tlsConfig}

	return mgr, nil
}
