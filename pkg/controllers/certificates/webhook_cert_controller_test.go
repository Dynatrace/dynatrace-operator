package certificates

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	res, err := controller.Reconcile(t.Context(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(t.Context(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
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

	res, err := controller.Reconcile(t.Context(), request)
	require.Error(t, err)
	assert.Equal(t, reconcile.Result{}, res)
}

func TestReconcileCertificate_Update(t *testing.T) {
	clt := newFakeClientBuilder().WithInvalidCertificateSecret().WithCRD().Build()
	controller, request := prepareController(clt)

	res, err := controller.Reconcile(t.Context(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(t.Context(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
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

	res, err := controller.Reconcile(t.Context(), request)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, SuccessDuration, res.RequeueAfter)

	secret := &corev1.Secret{}
	err = clt.Get(t.Context(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
	require.NoError(t, err)

	verifyCertificates(t, controller, secret, clt, false)
}

func TestReconcile(t *testing.T) {
	dkCrd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8scrd.DynaKubeName,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Conversion: &apiextensionsv1.CustomResourceConversion{
				Strategy: strategyWebhook,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: &apiextensionsv1.WebhookClientConfig{},
				},
			},
		},
	}

	ecCrd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8scrd.EdgeConnectName,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Conversion: &apiextensionsv1.CustomResourceConversion{
				Strategy: strategyWebhook,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: &apiextensionsv1.WebhookClientConfig{},
				},
			},
		},
	}

	t.Run("reconcile successfully without mutatingwebhookconfiguration", func(t *testing.T) {
		fakeClient := fake.NewClient(dkCrd, ecCrd,
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
		result, err := controller.Reconcile(t.Context(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("reconcile successfully without validatingwebhookconfiguration", func(t *testing.T) {
		fakeClient := fake.NewClient(dkCrd, ecCrd,
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
		result, err := controller.Reconcile(t.Context(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("update crd successfully with up-to-date secret", func(t *testing.T) {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		}
		fakeClient := fake.NewClient(dkCrd, ecCrd, deployment)
		cs := newCertificateSecret(deployment)
		_ = cs.setSecretFromReader(t.Context(), fakeClient, testNamespace)
		_ = cs.validateCertificates(testNamespace)
		_ = cs.createOrUpdateIfNecessary(t.Context(), fakeClient)

		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(t.Context(), request)
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedBundle, err := cs.loadCombinedBundle()
		require.NoError(t, err)

		actualCrd := &apiextensionsv1.CustomResourceDefinition{}
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: k8scrd.DynaKubeName}, actualCrd)
		require.NoError(t, err)
		assert.Equal(t, expectedBundle, actualCrd.Spec.Conversion.Webhook.ClientConfig.CABundle)
	})

	// Generation must not be skipped because webhook startup routine listens for the secret
	// See cmd/operator/manager.go and cmd/operator/watcher.go
	t.Run("do not skip certificates generation if no configuration exists", func(t *testing.T) {
		fakeClient := fake.NewClient(dkCrd, ecCrd, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.DeploymentName,
				Namespace: testNamespace,
			},
		})
		controller, request := prepareController(fakeClient)
		result, err := controller.Reconcile(t.Context(), request)

		require.NoError(t, err)
		assert.NotNil(t, result)

		secret := &corev1.Secret{}
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: expectedSecretName, Namespace: testNamespace}, secret)
		require.NoError(t, err)
	})
}

func createInvalidTestCertData() map[string][]byte {
	return map[string][]byte{
		RootKey:    {testBytes},
		RootCert:   {testBytes},
		ServerKey:  {testBytes},
		ServerCert: {testBytes},
	}
}

func createValidTestCertData() map[string][]byte {
	cert := Certs{
		Domain: testDomain,
		Now:    time.Now(),
	}
	_ = cert.ValidateCerts()

	return cert.Data
}

func createTestSecret(certData map[string][]byte) *corev1.Secret {
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
	}

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      webhook.DeploymentName,
			Namespace: testNamespace,
		},
	}

	return rec, request
}

func verifyCertificates(t *testing.T, rec *WebhookCertificateController, secret *corev1.Secret, clt client.Client, isUpdate bool) {
	t.Helper()
	cert := Certs{
		Domain:  getDomain(rec.namespace),
		Data:    secret.Data,
		SrcData: secret.Data,
		Now:     time.Now(),
	}

	// validateRootCerts and validateServerCerts return false if the certificates are valid
	assert.False(t, cert.validateRootCerts(time.Now()))
	assert.False(t, cert.validateServerCerts(time.Now()))

	assertCABundle := func(t *testing.T, webhookClientConfig *admissionregistrationv1.WebhookClientConfig) {
		t.Helper()
		require.NotNil(t, webhookClientConfig)
		require.NotEmpty(t, webhookClientConfig.CABundle)

		expectedCert := secret.Data[RootCert]
		if isUpdate {
			expectedCert = append(expectedCert, []byte{123}...)
		}

		assert.Equal(t, expectedCert, webhookClientConfig.CABundle)
	}

	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err := clt.Get(t.Context(), client.ObjectKey{Name: webhook.DeploymentName}, mutatingWebhookConfig)
	require.NoError(t, err)
	assert.Len(t, mutatingWebhookConfig.Webhooks, 2)
	assertCABundle(t, &mutatingWebhookConfig.Webhooks[0].ClientConfig)

	validatingWebhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	err = clt.Get(t.Context(), client.ObjectKey{Name: webhook.DeploymentName}, validatingWebhookConfig)
	require.NoError(t, err)
	assert.Len(t, validatingWebhookConfig.Webhooks, 1)
	assertCABundle(t, &validatingWebhookConfig.Webhooks[0].ClientConfig)
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
	builder.objs = append(builder.objs, createTestSecret(createValidTestCertData()))

	return builder
}

func (builder *fakeClientBuilder) WithInvalidCertificateSecret() *fakeClientBuilder {
	builder.objs = append(builder.objs, createTestSecret(createInvalidTestCertData()))

	return builder
}

func (builder *fakeClientBuilder) WithCRD() *fakeClientBuilder {
	builder.objs = append(builder.objs,
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8scrd.DynaKubeName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Conversion: &apiextensionsv1.CustomResourceConversion{
					Strategy: strategyWebhook,
					Webhook: &apiextensionsv1.WebhookConversion{
						ClientConfig: &apiextensionsv1.WebhookClientConfig{},
					},
				},
			},
		},
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: k8scrd.EdgeConnectName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Conversion: &apiextensionsv1.CustomResourceConversion{
					Strategy: strategyWebhook,
					Webhook: &apiextensionsv1.WebhookConversion{
						ClientConfig: &apiextensionsv1.WebhookClientConfig{},
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
