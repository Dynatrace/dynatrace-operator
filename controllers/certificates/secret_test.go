package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testNamespace = "test-namespace"
)

func TestGetSecret(t *testing.T) {
	t.Run(`get nil if secret does not exists`, func(t *testing.T) {
		clt := fake.NewClient()
		r := &ReconcileWebhookCertificates{
			client: clt,
			ctx:    context.TODO(),
		}
		secret, err := r.getSecret()
		require.NoError(t, err)
		assert.Nil(t, secret)
	})
	t.Run(`get secret`, func(t *testing.T) {
		clt := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhookDeploymentName + "-certs",
				Namespace: testNamespace,
			},
		})
		r := &ReconcileWebhookCertificates{
			client:    clt,
			ctx:       context.TODO(),
			namespace: testNamespace,
		}
		secret, err := r.getSecret()
		require.NoError(t, err)
		assert.NotNil(t, secret)
	})
}

func TestCertificateReconciler_ReconcileCertificateSecretForWebhook(t *testing.T) {
	const expectedSecretName = webhookDeploymentName + "-certs"

	t.Run(`create new certificates`, func(t *testing.T) {
		clt := fake.NewClient(
			&admissionregistrationv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webhookDeploymentName,
					Namespace: testNamespace,
				},
				Webhooks: []admissionregistrationv1.MutatingWebhook{
					{
						ClientConfig: admissionregistrationv1.WebhookClientConfig{},
					},
				},
			},
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webhookDeploymentName,
					Namespace: testNamespace,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						ClientConfig: admissionregistrationv1.WebhookClientConfig{},
					},
				},
			},
		)
		r := &ReconcileWebhookCertificates{
			ctx:       context.TODO(),
			client:    clt,
			namespace: testNamespace,
			logger:    logger.NewDTLogger(),
		}

		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      webhookDeploymentName,
				Namespace: testNamespace,
			},
		}

		res, err := r.Reconcile(context.TODO(), request)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, SuccessDuration, res.RequeueAfter)

		var secret corev1.Secret
		err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, &secret)

		assert.NoError(t, err)
		assert.NotNil(t, secret.Data)
		assert.NotEmpty(t, secret.Data)
		assert.Contains(t, secret.Data, certificate)
		assert.Contains(t, secret.Data, oldCertificate)
		assert.NotNil(t, secret.Data[certificate])
		assert.NotEmpty(t, secret.Data[certificate])
		assert.Empty(t, secret.Data[oldCertificate])

		cert := Certs{
			Log:     r.logger,
			Domain:  r.getDomain(),
			Data:    secret.Data,
			SrcData: secret.Data,
			Now:     time.Now(),
		}

		// validateRootCerts and validateServerCerts return false if the certificates are valid
		assert.False(t, cert.validateRootCerts(time.Now()))
		assert.False(t, cert.validateServerCerts(time.Now()))

		mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{}
		err = clt.Get(context.TODO(), client.ObjectKey{
			Name:      webhookDeploymentName,
			Namespace: testNamespace,
		}, mutatingWebhookConfig)
		require.NoError(t, err)
		assert.Len(t, mutatingWebhookConfig.Webhooks, 1)
		testWebhookClientConfig(t, mutatingWebhookConfig.Webhooks[0].ClientConfig, secret.Data)

		validationWebhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{}
		err = clt.Get(context.TODO(), client.ObjectKey{
			Name:      webhookDeploymentName,
			Namespace: testNamespace,
		}, validationWebhookConfig)
		require.NoError(t, err)
		testWebhookClientConfig(t, validationWebhookConfig.Webhooks[0].ClientConfig, secret.Data)
	})
	//	t.Run(`update certificates`, func(t *testing.T) {
	//		oldTime, err := time.Parse(time.RFC3339, "2011-01-01T00:00:00Z")
	//		require.NoError(t, err)
	//
	//		clt := fake.NewClient()
	//		r := &CertificateReconciler{
	//			ctx:         context.TODO(),
	//			clt:         clt,
	//			webhookName: testName,
	//			namespace:   testNamespace,
	//			logger:      logger.NewDTLogger(),
	//		}
	//		webhookConfig := &admissionregistrationv1.WebhookClientConfig{}
	//		cert := Certs{
	//			Log:    r.logger,
	//			Domain: r.getDomain(),
	//			Now:    oldTime,
	//			Data:   make(map[string][]byte),
	//		}
	//
	//		err = cert.generateRootCerts(r.getDomain(), oldTime)
	//		require.NoError(t, err)
	//		err = cert.generateServerCerts(r.getDomain(), oldTime)
	//		require.NoError(t, err)
	//
	//		secret := &v1.Secret{
	//			ObjectMeta: metav1.ObjectMeta{
	//				Name:      expectedSecretName,
	//				Namespace: testNamespace,
	//			},
	//			Data: cert.Data,
	//		}
	//		err = clt.Create(context.TODO(), secret)
	//		require.NoError(t, err)
	//
	//		err = r.ReconcileCertificateSecretForWebhook(webhookConfig)
	//
	//		assert.NoError(t, err)
	//
	//		err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	//
	//		assert.NoError(t, err)
	//		assert.NoError(t, err)
	//		assert.NotNil(t, secret.Data)
	//		assert.NotEmpty(t, secret.Data)
	//		assert.Contains(t, secret.Data, certificate)
	//		assert.Contains(t, secret.Data, oldCertificate)
	//		assert.NotNil(t, secret.Data[certificate])
	//		assert.NotNil(t, secret.Data[oldCertificate])
	//		assert.NotEmpty(t, secret.Data[certificate])
	//		assert.NotEmpty(t, secret.Data[oldCertificate])
	//		assert.NotNil(t, webhookConfig.CABundle)
	//		assert.NotEmpty(t, webhookConfig.CABundle)
	//		assert.Equal(t, append(secret.Data[certificate], secret.Data[oldCertificate]...), webhookConfig.CABundle)
	//	})
}

//
//func TestReconcileWebhookCertificates(t *testing.T) {
//	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout))
//	ns := "dynatrace"
//
//	tmpDir, err := ioutil.TempDir("", "webhook-certs")
//	require.NoError(t, err)
//	defer func() { _ = os.RemoveAll(tmpDir) }()
//
//	c := fake.NewClient(&admissionregistrationv1.MutatingWebhookConfiguration{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: webhookName,
//		},
//		Webhooks: []admissionregistrationv1.MutatingWebhook{
//			{
//				Name: webhookName,
//			},
//		},
//	},
//		&corev1.Service{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      webhookName,
//				Namespace: ns,
//			},
//		})
//	r := ReconcileWebhookCertificates{client: c, logger: logger}
//
//	_, err = r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}})
//	require.NoError(t, err)
//
//	var secret corev1.Secret
//	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.SecretCertsName, Namespace: ns}, &secret))
//
//	m := make(map[string]string, len(secret.Data))
//	for k, v := range secret.Data {
//		m[k] = string(v)
//	}
//
//	getWebhookCA := func() string {
//		var webhookCfg admissionregistrationv1.MutatingWebhookConfiguration
//		require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName}, &webhookCfg))
//		return string(webhookCfg.Webhooks[0].ClientConfig.CABundle)
//	}
//
//	var service corev1.Service
//	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Name: webhook.ServiceName, Namespace: ns}, &service))
//
//	assert.NotEmpty(t, secret.Data["tls.crt"])
//	assert.NotEmpty(t, secret.Data["tls.key"])
//	assert.NotEmpty(t, secret.Data["ca.crt"])
//	assert.NotEmpty(t, secret.Data["ca.key"])
//	assert.Equal(t, "", string(secret.Data["ca.crt.old"]))
//	assert.Equal(t, getWebhookCA(), string(secret.Data["ca.crt"]))
//}

func testWebhookClientConfig(t *testing.T, webhookClientConfig admissionregistrationv1.WebhookClientConfig, secretData map[string][]byte) {
	assert.NotNil(t, webhookClientConfig)
	assert.NotNil(t, webhookClientConfig.CABundle)
	assert.NotEmpty(t, webhookClientConfig.CABundle)
	assert.Equal(t, secretData[certificate], webhookClientConfig.CABundle)
}
