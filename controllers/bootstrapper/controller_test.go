package bootstrapper

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileWebhook(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))
	ns := "dynatrace"

	tmpDir, err := ioutil.TempDir("", "webhook-certs")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	now, err := time.Parse(time.RFC3339, "2018-01-10T00:00:00Z")
	require.NoError(t, err)

	c := fake.NewClient()
	r := ReconcileWebhook{client: c, logger: logger, namespace: ns, scheme: scheme.Scheme, certsDir: tmpDir}

	reconcileAndGetCreds := func(days time.Duration) map[string]string {
		r.now = now.Add(days * 24 * time.Hour)
		_, err = r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}})
		require.NoError(t, err)

		var secret corev1.Secret
		require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretCertsName, Namespace: ns}, &secret))

		m := make(map[string]string, len(secret.Data))
		for k, v := range secret.Data {
			m[k] = string(v)
		}
		return m
	}

	getWebhookCA := func() string {
		var webhookCfg admissionregistrationv1beta1.MutatingWebhookConfiguration
		require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName}, &webhookCfg))
		return string(webhookCfg.Webhooks[0].ClientConfig.CABundle)
	}

	// Day 0: No objects exist, create them.

	secret0 := reconcileAndGetCreds(0)

	var service corev1.Service
	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}, &service))

	assert.NotEmpty(t, secret0["tls.crt"])
	assert.NotEmpty(t, secret0["tls.key"])
	assert.NotEmpty(t, secret0["ca.crt"])
	assert.NotEmpty(t, secret0["ca.key"])
	assert.Equal(t, secret0["ca.crt"], getWebhookCA())

	// Day 1: Certificates are valid, no changes.

	secret1 := reconcileAndGetCreds(1)
	assert.Equal(t, secret0, secret1)
	assert.Equal(t, secret1["ca.crt"], getWebhookCA())

	// Day 8: TLS certificates have expired and need to be renewed.

	secret8 := reconcileAndGetCreds(8)
	assert.NotEqual(t, secret1["tls.crt"], secret8["tls.crt"])
	assert.NotEqual(t, secret1["tls.key"], secret8["tls.key"])
	assert.Equal(t, secret1["ca.crt"], secret8["ca.crt"])
	assert.Equal(t, secret1["ca.key"], secret8["ca.key"])
	assert.Equal(t, secret8["ca.crt"], getWebhookCA())

	// Day 9: TLS certificates were renewed recently, no changes.

	secret9 := reconcileAndGetCreds(9)
	assert.Equal(t, secret8, secret9)
	assert.Equal(t, secret9["ca.crt"], getWebhookCA())

	// Day 400: CA certificates have expired and both TLS and CA certs need to be renewed.

	secret400 := reconcileAndGetCreds(400)
	assert.NotEqual(t, secret9["tls.crt"], secret400["tls.crt"])
	assert.NotEqual(t, secret9["tls.key"], secret400["tls.key"])
	assert.NotEqual(t, secret9["ca.crt"], secret400["ca.crt"])
	assert.NotEqual(t, secret9["ca.key"], secret400["ca.key"])
	assert.Equal(t, secret400["ca.crt"], getWebhookCA())

	// Day 401: CA and TLS certificates were renewed recently, no changes.

	secret401 := reconcileAndGetCreds(401)
	assert.Equal(t, secret400, secret401)
	assert.Equal(t, secret401["ca.crt"], getWebhookCA())
}
