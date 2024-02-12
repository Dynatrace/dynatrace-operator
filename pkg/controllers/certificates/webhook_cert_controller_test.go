package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	strategyWebhook = "webhook"
)

func TestReconcileCertificate_Create(t *testing.T) {
	clt := newFakeClientBuilder().WithCRD().Build()
	controller, request := prepareController(clt)

	res, err := controller.Reconcile(context.TODO(), request)
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

	verifyCertificates(t, controller, secret, clt, false)
}

func TestReconcileCertificate_Create_NoCRD(t *testing.T) {
	clt := newFakeClientBuilder().Build()
	controller, request := prepareController(clt)

	res, err := controller.Reconcile(context.TODO(), request)
	require.Error(t, err)
	assert.Equal(t, reconcile.Result{}, res)
}

func TestReconcileCertificate_Update(t *testing.T) {
	clt := newFakeClientBuilder().WithInvalidCertificateSecret().WithCRD().Build()
	controller, request := prepareController(clt)

	res, err := controller.Reconcile(context.TODO(), request)
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

	verifyCertificates(t, controller, secret, clt, true)
}

func TestReconcileCertificate_ExistingSecretWithValidCertificate(t *testing.T) {
	clt := newFakeClientBuilder().WithValidCertificateSecret().WithCRD().Build()
	controller, request := prepareController(clt)

	res, err := controller.Reconcile(context.TODO(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	require.NoError(t, err)

	verifyCertificates(t, controller, secret, clt, false)
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
		fakeClient := fake.NewClient(crd,
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
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webhook.DeploymentName,
					Namespace: testNamespace,
				},
			})
		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(context.TODO(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run(`reconcile successfully without validatingwebhookconfiguration`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd,
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
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      webhook.DeploymentName,
					Namespace: testNamespace,
				},
			})
		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(context.TODO(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run(`update crd successfully with up-to-date secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		})
		cs := newCertificateSecret(scheme.Scheme, &appsv1.Deployment{})
		_ = cs.setSecretFromReader(context.TODO(), fakeClient, testNamespace)
		_ = cs.validateCertificates(testNamespace)
		_ = cs.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(context.TODO(), request)
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedBundle, err := cs.loadCombinedBundle()
		require.NoError(t, err)

		actualCrd := &apiv1.CustomResourceDefinition{}
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: crdName}, actualCrd)
		require.NoError(t, err)
		assert.Equal(t, expectedBundle, actualCrd.Spec.Conversion.Webhook.ClientConfig.CABundle)
	})

	// Generation must not be skipped because webhook startup routine listens for the secret
	// See cmd/operator/manager.go and cmd/operator/watcher.go
	t.Run(`do not skip certificates generation if no configuration exists`, func(t *testing.T) {
		fakeClient := fake.NewClient(crd, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		})
		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(context.TODO(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)

		secret := &corev1.Secret{}
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
		require.NoError(t, err)
	})
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

func prepareController(clt client.Client) (*WebhookCertificateController, reconcile.Request) {
	rec := &WebhookCertificateController{
		client:    clt,
		apiReader: clt,
		namespace: testNamespace,
		scheme:    scheme.Scheme,
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

func verifyCertificates(t *testing.T, rec *WebhookCertificateController, secret *corev1.Secret, clt client.Client, isUpdate bool) {
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

type fakeClientBuilder struct {
	objs []client.Object
}

func newFakeClientBuilder() *fakeClientBuilder {
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
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		},
	}

	return &fakeClientBuilder{objs: objs}
}

func (builder *fakeClientBuilder) WithValidCertificateSecret() *fakeClientBuilder {
	builder.objs = append(builder.objs,
		createTestSecret(nil, createValidTestCertData(nil)),
	)

	return builder
}

func (builder *fakeClientBuilder) WithInvalidCertificateSecret() *fakeClientBuilder {
	builder.objs = append(builder.objs,
		createTestSecret(nil, createInvalidTestCertData(nil)),
	)

	return builder
}

func (builder *fakeClientBuilder) WithCRD() *fakeClientBuilder {
	builder.objs = append(builder.objs,
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
	)

	return builder
}

func (builder *fakeClientBuilder) Build() client.Client {
	return fake.NewClient(builder.objs...)
}
