package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
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
	testDomain    = webhook.DeploymentName + "." + testNamespace + ".svc"

	expectedSecretName = webhook.DeploymentName + secretPostfix

	testBytes = 123
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
				Name:      expectedSecretName,
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

func TestReconcileCertificate_Create(t *testing.T) {
	clt := prepareFakeClient(false, false)
	rec, request := prepareReconcile(clt)

	res, err := rec.Reconcile(context.TODO(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	require.NoError(t, err)

	assert.NotNil(t, secret.Data)
	assert.NotEmpty(t, secret.Data)
	assert.Contains(t, secret.Data, RootKey)
	assert.Contains(t, secret.Data, RootCert)
	assert.Contains(t, secret.Data, RootCertOld)
	assert.Contains(t, secret.Data, ServerKey)
	assert.Contains(t, secret.Data, ServerCert)
	assert.NotNil(t, secret.Data[RootCert])
	assert.NotEmpty(t, secret.Data[RootCert])
	assert.Empty(t, secret.Data[RootCertOld])

	verifyCertificates(t, rec, secret, clt, false)
}

func TestReconcileCertificate_Update(t *testing.T) {
	clt := prepareFakeClient(true, false)
	rec, request := prepareReconcile(clt)

	res, err := rec.Reconcile(context.TODO(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	require.NoError(t, err)

	assert.NotNil(t, secret.Data)
	assert.NotEmpty(t, secret.Data)
	assert.Contains(t, secret.Data, RootKey)
	assert.Contains(t, secret.Data, RootCert)
	assert.Contains(t, secret.Data, RootCertOld)
	assert.Contains(t, secret.Data, ServerKey)
	assert.Contains(t, secret.Data, ServerCert)
	assert.NotNil(t, secret.Data[RootCert])
	assert.NotEmpty(t, secret.Data[RootCert])
	assert.Equal(t, []byte{testBytes}, secret.Data[RootCertOld])

	verifyCertificates(t, rec, secret, clt, true)
}

func TestReconcileCertificate_ExistingSecretWithValidCertificate(t *testing.T) {
	clt := prepareFakeClient(true, true)
	rec, request := prepareReconcile(clt)

	res, err := rec.Reconcile(context.TODO(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	require.NoError(t, err)

	verifyCertificates(t, rec, secret, clt, false)
}

func prepareFakeClient(withSecret bool, generateValidSecret bool) client.Client {
	objs := []client.Object{
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhook.DeploymentName,
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				{
					ClientConfig: admissionregistrationv1.WebhookClientConfig{},
				},
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhook.DeploymentName,
			},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					ClientConfig: admissionregistrationv1.WebhookClientConfig{},
				},
			},
		},
	}
	if withSecret {
		certData := map[string][]byte{
			RootKey:    {testBytes},
			RootCert:   {testBytes},
			ServerKey:  {testBytes},
			ServerCert: {testBytes},
		}
		if generateValidSecret {
			cert := Certs{
				Log:    logger.NewDTLogger(),
				Domain: testDomain,
				Now:    time.Now(),
			}
			_ = cert.ValidateCerts()

			certData = cert.Data
		}

		objs = append(objs,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      expectedSecretName,
				},
				Data: certData,
			},
		)
	}

	return fake.NewClient(objs...)
}

func prepareReconcile(clt client.Client) (*ReconcileWebhookCertificates, reconcile.Request) {
	rec := &ReconcileWebhookCertificates{
		ctx:       context.TODO(),
		client:    clt,
		namespace: testNamespace,
		logger:    logger.NewDTLogger(),
	}

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      webhook.DeploymentName,
			Namespace: testNamespace,
		},
	}

	return rec, request
}

func testWebhookClientConfig(
	t *testing.T, webhookClientConfig *admissionregistrationv1.WebhookClientConfig,
	secretData map[string][]byte, isUpdate bool) {
	require.NotNil(t, webhookClientConfig)
	require.NotEmpty(t, webhookClientConfig.CABundle)

	expectedCert := secretData[RootCert]
	if isUpdate {
		expectedCert = append(expectedCert, []byte{123}...)
	}
	assert.Equal(t, expectedCert, webhookClientConfig.CABundle)
}

func verifyCertificates(t *testing.T, rec *ReconcileWebhookCertificates, secret *corev1.Secret, clt client.Client, isUpdate bool) {
	cert := Certs{
		Log:     rec.logger,
		Domain:  rec.getDomain(),
		Data:    secret.Data,
		SrcData: secret.Data,
		Now:     time.Now(),
	}

	// validateRootCerts and validateServerCerts return false if the certificates are valid
	assert.False(t, cert.validateRootCerts(time.Now()))
	assert.False(t, cert.validateServerCerts(time.Now()))

	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err := clt.Get(context.TODO(), client.ObjectKey{
		Name: webhook.DeploymentName,
	}, mutatingWebhookConfig)
	require.NoError(t, err)
	assert.Len(t, mutatingWebhookConfig.Webhooks, 1)
	testWebhookClientConfig(t, &mutatingWebhookConfig.Webhooks[0].ClientConfig, secret.Data, isUpdate)

	validatingWebhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	err = clt.Get(context.TODO(), client.ObjectKey{
		Name: webhook.DeploymentName,
	}, validatingWebhookConfig)
	require.NoError(t, err)
	assert.Len(t, validatingWebhookConfig.Webhooks, 1)
	testWebhookClientConfig(t, &validatingWebhookConfig.Webhooks[0].ClientConfig, secret.Data, isUpdate)
}
