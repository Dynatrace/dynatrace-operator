package certificates

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName = "test-name"
	testKey  = "test-key"
)

var testValue1 = []byte{1, 2, 3, 4}
var testValue2 = []byte{5, 6, 7, 8}

func TestSetSecretFromReader(t *testing.T) {
	t.Run(`fill with empty secret if secret does not exist`, func(t *testing.T) {
		certSecret := newCertificateSecret()
		err := certSecret.setSecretFromReader(context.TODO(), fake.NewClient(), testNamespace)

		assert.NoError(t, err)
		assert.False(t, certSecret.existsInCluster)
		assert.NotNil(t, certSecret.secret)
	})
	t.Run(`find existing secret`, func(t *testing.T) {
		certSecret := newCertificateSecret()
		err := certSecret.setSecretFromReader(context.TODO(), fake.NewClient(
			createTestSecret(t, createInvalidTestCertData(t))), testNamespace)

		assert.NoError(t, err)
		assert.True(t, certSecret.existsInCluster)
		assert.NotNil(t, certSecret.secret)
	})
}

func TestIsRecent(t *testing.T) {
	t.Run(`true if certs and secret are nil`, func(t *testing.T) {
		certSecret := newCertificateSecret()

		assert.True(t, certSecret.isRecent())
	})
	t.Run(`false if only one is nil`, func(t *testing.T) {
		certSecret := newCertificateSecret()
		certSecret.secret = &corev1.Secret{}

		assert.False(t, certSecret.isRecent())

		certSecret.secret = nil
		certSecret.certificates = &Certs{}

		assert.False(t, certSecret.isRecent())
	})
	t.Run(`true if data is equal, false otherwise`, func(t *testing.T) {
		certSecret := newCertificateSecret()
		secret := corev1.Secret{
			Data: map[string][]byte{testKey: testValue1},
		}
		certs := Certs{
			Data: map[string][]byte{testKey: testValue1},
		}
		certSecret.secret = &secret
		certSecret.certificates = &certs

		assert.True(t, certSecret.isRecent())

		certSecret.secret.Data = map[string][]byte{testKey: testValue2}

		assert.False(t, certSecret.isRecent())
	})
}

func TestAreConfigsValid(t *testing.T) {
	t.Run(`true if no configs were given`, func(t *testing.T) {
		certSecret := newCertificateSecret()

		assert.True(t, certSecret.areConfigsValid(nil))
		assert.True(t, certSecret.areConfigsValid(make([]*admissionregistrationv1.WebhookClientConfig, 0)))
	})
	t.Run(`true if all CABundle matches certificate data, false otherwise`, func(t *testing.T) {
		certSecret := newCertificateSecret()
		certSecret.certificates = &Certs{
			Data: map[string][]byte{RootCert: testValue1},
		}
		webhookConfigs := make([]*admissionregistrationv1.WebhookClientConfig, 1)
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})

		assert.True(t, certSecret.areConfigsValid(webhookConfigs))

		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue2,
		})

		assert.False(t, certSecret.areConfigsValid(webhookConfigs))
	})
}

func TestCreateOrUpdateIfNecessary(t *testing.T) {
	t.Run(`do nothing if certificate is recent and exists`, func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret()
		certSecret.existsInCluster = true

		err := certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		assert.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName()}, &corev1.Secret{})

		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run(`create if secret does not exist`, func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret()
		certSecret.secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildSecretName(),
				Namespace: testNamespace,
			},
		}
		certSecret.certificates = &Certs{
			Data: map[string][]byte{testKey: testValue1},
		}

		err := certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		assert.NoError(t, err)

		newSecret := corev1.Secret{}
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName(), Namespace: testNamespace}, &newSecret)

		assert.NoError(t, err)
		assert.NotNil(t, newSecret)
		assert.EqualValues(t, certSecret.certificates.Data, newSecret.Data)
	})
	t.Run(`update if secret exists`, func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret()
		certSecret.secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildSecretName(),
				Namespace: testNamespace,
			},
		}
		certSecret.certificates = &Certs{
			Data: map[string][]byte{testKey: testValue1},
		}

		err := certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		require.NoError(t, err)

		newSecret := corev1.Secret{}
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName(), Namespace: testNamespace}, &newSecret)

		require.NoError(t, err)
		require.NotNil(t, newSecret)
		require.EqualValues(t, certSecret.certificates.Data, newSecret.Data)

		certSecret.secret = &newSecret
		certSecret.certificates.Data = map[string][]byte{testKey: testValue2}
		certSecret.existsInCluster = true
		err = certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		assert.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName(), Namespace: testNamespace}, &newSecret)

		assert.NoError(t, err)
		assert.NotNil(t, newSecret)
		assert.EqualValues(t, certSecret.certificates.Data, newSecret.Data)
	})
}

func TestUpdateClientConfigurations(t *testing.T) {
	mutatingWebhookConfig := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{},
			},
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{},
			},
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{},
			},
		},
	}
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildSecretName(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{RootCert: testValue1},
	}
	fakeClient := fake.NewClient(&mutatingWebhookConfig, &secret)
	clientConfigs := getClientConfigsFromMutatingWebhook(&mutatingWebhookConfig)
	certSecret := newCertificateSecret()
	certSecret.secret = &secret

	err := certSecret.updateClientConfigurations(context.TODO(), fakeClient, clientConfigs, &mutatingWebhookConfig)

	assert.NoError(t, err)

	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testName, Namespace: testNamespace}, &mutatingWebhookConfig)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(mutatingWebhookConfig.Webhooks))

	for _, mutatingWebhook := range mutatingWebhookConfig.Webhooks {
		assert.EqualValues(t, mutatingWebhook.ClientConfig.CABundle, secret.Data[RootCert])
	}
}
