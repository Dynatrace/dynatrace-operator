package webhook

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileWebhookCertificates(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))
	ns := "dynatrace"

	tmpDir, err := ioutil.TempDir("", "webhook-certs")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := fake.NewClient(&admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: webhookName,
			},
		},
	},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhookName,
				Namespace: ns,
			},
		})
	r := ReconcileWebhookCertificates{client: c, logger: logger, namespace: ns, scheme: scheme.Scheme}

	_, err = r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}})
	require.NoError(t, err)

	var secret corev1.Secret
	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretCertsName, Namespace: ns}, &secret))

	m := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		m[k] = string(v)
	}

	getWebhookCA := func() string {
		var webhookCfg admissionregistrationv1.MutatingWebhookConfiguration
		require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName}, &webhookCfg))
		return string(webhookCfg.Webhooks[0].ClientConfig.CABundle)
	}

	var service corev1.Service
	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}, &service))

	assert.NotEmpty(t, secret.Data["tls.crt"])
	assert.NotEmpty(t, secret.Data["tls.key"])
	assert.NotEmpty(t, secret.Data["ca.crt"])
	assert.NotEmpty(t, secret.Data["ca.key"])
	assert.Equal(t, "", string(secret.Data["ca.crt.old"]))
	assert.Equal(t, getWebhookCA(), string(secret.Data["ca.crt"]))
}
