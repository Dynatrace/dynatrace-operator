package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestGetSecret(t *testing.T) {
	t.Run(`get nil if secret does not exists`, func(t *testing.T) {
		clt := fake.NewClient()
		r := &CertificateReconciler{
			clt: clt,
			ctx: context.TODO(),
		}
		secret, err := r.getSecret()
		assert.NoError(t, err)
		assert.Nil(t, secret)
	})
	t.Run(`get secret`, func(t *testing.T) {
		clt := fake.NewClient(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "-certs",
				Namespace: testNamespace,
			},
		})
		r := &CertificateReconciler{
			clt:         clt,
			ctx:         context.TODO(),
			webhookName: testName,
			namespace:   testNamespace,
		}
		secret, err := r.getSecret()
		assert.NoError(t, err)
		assert.NotNil(t, secret)
	})
}

func TestCertificateReconciler_ReconcileCertificateSecretForWebhook(t *testing.T) {
	const expectedSecretName = testName + "-certs"

	t.Run(`create new certificates`, func(t *testing.T) {
		clt := fake.NewClient()
		r := &CertificateReconciler{
			ctx:         context.TODO(),
			clt:         clt,
			webhookName: testName,
			namespace:   testNamespace,
			logger:      logger.NewDTLogger(),
		}
		webhookConfig := &admissionv1.WebhookClientConfig{}
		err := r.ReconcileCertificateSecretForWebhook(webhookConfig)

		assert.NoError(t, err)

		var secret v1.Secret
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

		assert.NotNil(t, webhookConfig.CABundle)
		assert.NotEmpty(t, webhookConfig.CABundle)
		assert.Equal(t, secret.Data[certificate], webhookConfig.CABundle)
	})
	t.Run(`update certificates`, func(t *testing.T) {
		oldTime, err := time.Parse(time.RFC3339, "2011-01-01T00:00:00Z")
		require.NoError(t, err)

		clt := fake.NewClient()
		r := &CertificateReconciler{
			ctx:         context.TODO(),
			clt:         clt,
			webhookName: testName,
			namespace:   testNamespace,
			logger:      logger.NewDTLogger(),
		}
		webhookConfig := &admissionv1.WebhookClientConfig{}
		cert := Certs{
			Log:    r.logger,
			Domain: r.getDomain(),
			Now:    oldTime,
			Data:   make(map[string][]byte),
		}

		err = cert.generateRootCerts(r.getDomain(), oldTime)
		require.NoError(t, err)
		err = cert.generateServerCerts(r.getDomain(), oldTime)
		require.NoError(t, err)

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      expectedSecretName,
				Namespace: testNamespace,
			},
			Data: cert.Data,
		}
		err = clt.Create(context.TODO(), secret)
		require.NoError(t, err)

		err = r.ReconcileCertificateSecretForWebhook(webhookConfig)

		assert.NoError(t, err)

		err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)

		assert.NoError(t, err)
		assert.NoError(t, err)
		assert.NotNil(t, secret.Data)
		assert.NotEmpty(t, secret.Data)
		assert.Contains(t, secret.Data, certificate)
		assert.Contains(t, secret.Data, oldCertificate)
		assert.NotNil(t, secret.Data[certificate])
		assert.NotNil(t, secret.Data[oldCertificate])
		assert.NotEmpty(t, secret.Data[certificate])
		assert.NotEmpty(t, secret.Data[oldCertificate])
		assert.NotNil(t, webhookConfig.CABundle)
		assert.NotEmpty(t, webhookConfig.CABundle)
		assert.Equal(t, append(secret.Data[certificate], secret.Data[oldCertificate]...), webhookConfig.CABundle)
	})
}
