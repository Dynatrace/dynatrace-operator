package webhook

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	managermock "github.com/Dynatrace/dynatrace-operator/test/mocks/sigs.k8s.io/controller-runtime/pkg/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestCreateOptions(t *testing.T) {
	t.Run("implements interface", func(t *testing.T) {
		var provider manager.Provider = NewProvider("certs-dir", "key-file", "cert-file")
		_, _ = provider.CreateManager("namespace", &rest.Config{})

		providerImpl := provider.(Provider)
		assert.Equal(t, "certs-dir", providerImpl.certificateDirectory)
		assert.Equal(t, "key-file", providerImpl.keyFileName)
		assert.Equal(t, "cert-file", providerImpl.certificateFileName)
	})
	t.Run("creates options", func(t *testing.T) {
		provider := Provider{}
		options := provider.createOptions("test-namespace")

		assert.NotNil(t, options)
		assert.Contains(t, options.Cache.DefaultNamespaces, "test-namespace")
		assert.Equal(t, scheme.Scheme, options.Scheme)
		assert.Equal(t, defaultMetricsBindAddress, options.Metrics.BindAddress)

		webhookServer, ok := options.WebhookServer.(*webhook.DefaultServer)
		require.True(t, ok)
		assert.Equal(t, defaultPort, webhookServer.Options.Port)
	})

	t.Run("creates options with custom ports", func(t *testing.T) {
		t.Setenv("WEBHOOK_PORT", "6443")
		t.Setenv("METRICS_BIND_ADDRESS", ":8081")
		t.Setenv("HEALTH_PROBE_BIND_ADDRESS", ":10081")

		provider := Provider{}
		options := provider.createOptions("test-namespace")

		assert.NotNil(t, options)
		assert.Contains(t, options.Cache.DefaultNamespaces, "test-namespace")
		assert.Equal(t, scheme.Scheme, options.Scheme)

		webhookServer, ok := options.WebhookServer.(*webhook.DefaultServer)
		require.True(t, ok)
		assert.Equal(t, 6443, webhookServer.Options.Port)
		assert.Equal(t, ":10081", options.HealthProbeBindAddress)
		assert.Equal(t, ":8081", options.Metrics.BindAddress)
	})
	t.Run("configures webhooks server", func(t *testing.T) {
		provider := NewProvider("certs-dir", "key-file", "cert-file")
		expectedWebhookServer := &webhook.DefaultServer{}

		mockedMgr := managermock.NewManager(t)
		mockedMgr.On("GetWebhookServer").Return(expectedWebhookServer)

		mgrWithWebhookServer, err := provider.setupWebhookServer(mockedMgr)
		require.NoError(t, err)

		mgrWebhookServer, ok := mgrWithWebhookServer.GetWebhookServer().(*webhook.DefaultServer)
		require.True(t, ok)
		assert.Equal(t, "certs-dir", mgrWebhookServer.Options.CertDir)
		assert.Equal(t, "key-file", mgrWebhookServer.Options.KeyName)
		assert.Equal(t, "cert-file", mgrWebhookServer.Options.CertName)
	})
}
