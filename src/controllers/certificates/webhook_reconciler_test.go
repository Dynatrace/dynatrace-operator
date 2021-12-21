package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	testNamespace = "test-namespace"
	testDomain    = webhook.DeploymentName + "." + testNamespace + ".svc"

	expectedSecretName = webhook.DeploymentName + secretPostfix

	testBytes = 123

	strategyWebhook = "webhook"
)

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

func TestReconcile(t *testing.T) {
	crd := &apiv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiv1.CustomResourceDefinitionSpec{
			Conversion: &apiv1.CustomResourceConversion{
				Strategy: strategyWebhook,
				Webhook: &apiv1.WebhookConversion{
					ClientConfig: &apiv1.WebhookClientConfig{},
				},
			},
		},
	}

	t.Run(`reconcile successfully without mutatingwebhookconfiguration`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd, &admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhook.DeploymentName,
			},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{
				{
					ClientConfig: admissionregistrationv1.WebhookClientConfig{},
				},
			},
		})
		reconciliation, request := prepareReconcile(fakeClient)
		result, err := reconciliation.Reconcile(context.TODO(), request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run(`reconcile successfully without validatingwebhookconfiguration`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd, &admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhook.DeploymentName,
			},
			Webhooks: []admissionregistrationv1.MutatingWebhook{
				{
					ClientConfig: admissionregistrationv1.WebhookClientConfig{},
				},
				{
					ClientConfig: admissionregistrationv1.WebhookClientConfig{},
				},
			},
		})
		reconciliation, request := prepareReconcile(fakeClient)
		result, err := reconciliation.Reconcile(context.TODO(), request)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run(`skip certificates generation if no configuration exists`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd)
		reconciliation, request := prepareReconcile(fakeClient)
		result, err := reconciliation.Reconcile(context.TODO(), request)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		secret := &corev1.Secret{}
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
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
		&apiv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdName,
			},
			Spec: apiv1.CustomResourceDefinitionSpec{
				Conversion: &apiv1.CustomResourceConversion{
					Strategy: strategyWebhook,
					Webhook: &apiv1.WebhookConversion{
						ClientConfig: &apiv1.WebhookClientConfig{},
					},
				},
			},
		},
	}
	if withSecret {
		certData := createInvalidTestCertData(nil)
		if generateValidSecret {
			certData = createValidTestCertData(nil)
		}

		objs = append(objs,
			createTestSecret(nil, certData),
		)
	}

	return fake.NewClient(objs...)
}

func createInvalidTestCertData(_ *testing.T) map[string][]byte {
	return map[string][]byte{
		RootKey:    {testBytes},
		RootCert:   {testBytes},
		ServerKey:  {testBytes},
		ServerCert: {testBytes},
	}
}

func createValidTestCertData(_ *testing.T) map[string][]byte {
	cert := Certs{
		Domain: testDomain,
		Now:    time.Now(),
	}
	_ = cert.ValidateCerts()
	return cert.Data
}

func createTestSecret(_ *testing.T, certData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      expectedSecretName,
		},
		Data: certData,
	}
}

func prepareReconcile(clt client.Client) (*ReconcileWebhookCertificates, reconcile.Request) {
	rec := &ReconcileWebhookCertificates{
		ctx:       context.TODO(),
		client:    clt,
		apiReader: clt,
		namespace: testNamespace,
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
		Domain:  getDomain(rec.namespace),
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
	assert.Len(t, mutatingWebhookConfig.Webhooks, 2)
	testWebhookClientConfig(t, &mutatingWebhookConfig.Webhooks[0].ClientConfig, secret.Data, isUpdate)

	validatingWebhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	err = clt.Get(context.TODO(), client.ObjectKey{
		Name: webhook.DeploymentName,
	}, validatingWebhookConfig)
	require.NoError(t, err)
	assert.Len(t, validatingWebhookConfig.Webhooks, 1)
	testWebhookClientConfig(t, &validatingWebhookConfig.Webhooks[0].ClientConfig, secret.Data, isUpdate)
}
