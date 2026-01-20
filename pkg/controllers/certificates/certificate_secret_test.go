package certificates

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testKey = "test-key"
)

var testValue1 = []byte{1, 2, 3, 4}
var testValue2 = []byte{5, 6, 7, 8}

func TestSetSecretFromReader(t *testing.T) {
	t.Run("fill with empty secret if secret does not exist", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})
		err := certSecret.setSecretFromReader(context.TODO(), fake.NewClient(), testNamespace)

		require.NoError(t, err)
		assert.False(t, certSecret.existsInCluster)
		assert.NotNil(t, certSecret.secret)
	})
	t.Run("find existing secret", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})
		err := certSecret.setSecretFromReader(context.TODO(), fake.NewClient(
			createTestSecret(createInvalidTestCertData())), testNamespace)

		require.NoError(t, err)
		assert.True(t, certSecret.existsInCluster)
		assert.NotNil(t, certSecret.secret)
	})
}

func TestIsRecent(t *testing.T) {
	t.Run("true if certs and secret are nil", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})

		assert.True(t, certSecret.isRecent())
	})
	t.Run("false if only one is nil", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})
		certSecret.secret = &corev1.Secret{}

		assert.False(t, certSecret.isRecent())

		certSecret.secret = nil
		certSecret.certificates = &Certs{}

		assert.False(t, certSecret.isRecent())
	})
	t.Run("true if data is equal, false otherwise", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})
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
	t.Run("true if no configs were given", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})

		assert.True(t, certSecret.areWebhookConfigsValid(nil))
		assert.True(t, certSecret.areWebhookConfigsValid(make([]*admissionregistrationv1.WebhookClientConfig, 0)))
	})
	t.Run("true if all CABundle matches certificate data, false otherwise", func(t *testing.T) {
		certSecret := newCertificateSecret(&appsv1.Deployment{})
		certSecret.certificates = &Certs{
			Data: map[string][]byte{RootCert: testValue1},
		}

		var webhookConfigs []*admissionregistrationv1.WebhookClientConfig
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})
		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue1,
		})

		assert.True(t, certSecret.areWebhookConfigsValid(webhookConfigs))

		webhookConfigs = append(webhookConfigs, &admissionregistrationv1.WebhookClientConfig{
			CABundle: testValue2,
		})

		assert.False(t, certSecret.areWebhookConfigsValid(webhookConfigs))
	})
}

func TestCreateOrUpdateIfNecessary(t *testing.T) {
	t.Run("do nothing if certificate is recent and exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret(&appsv1.Deployment{})
		certSecret.existsInCluster = true

		err := certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		require.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName()}, &corev1.Secret{})

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("create if secret does not exist", func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret(&appsv1.Deployment{})
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
		assert.NotNil(t, newSecret)
		assert.Equal(t, certSecret.certificates.Data, newSecret.Data)
	})
	t.Run("update if secret exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		certSecret := newCertificateSecret(&appsv1.Deployment{})
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
		require.Equal(t, certSecret.certificates.Data, newSecret.Data)

		certSecret.secret = &newSecret
		certSecret.certificates.Data = map[string][]byte{testKey: testValue2}
		certSecret.existsInCluster = true
		err = certSecret.createOrUpdateIfNecessary(context.TODO(), fakeClient)

		require.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: buildSecretName(), Namespace: testNamespace}, &newSecret)

		require.NoError(t, err)
		assert.NotNil(t, newSecret)
		assert.Equal(t, certSecret.certificates.Data, newSecret.Data)
	})
}

func TestCertificateSecret_isBundleValid(t *testing.T) {
	certSecret := &certificateSecret{
		certificates: &Certs{
			Data: map[string][]byte{RootCert: testValue1},
		},
	}

	tests := []struct {
		name string
		args []byte
		want bool
	}{
		{
			name: "bundle is nil",
			args: nil,
			want: false,
		},
		{
			name: "root cert and bundle are different",
			args: testValue2,
			want: false,
		},
		{
			name: "root cert and bundle are equal",
			args: testValue1,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, certSecret.isBundleValid(tt.args), "isBundleValid(%v)", tt.args)
		})
	}
}

func TestCertificateSecret_isCRDConversionValid(t *testing.T) {
	certSecret := &certificateSecret{
		certificates: &Certs{
			Data: map[string][]byte{RootCert: testValue1},
		},
	}

	tests := []struct {
		name string
		args *apiextensionsv1.CustomResourceDefinition
		want bool
	}{
		{
			name: "nil converter is valid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{}),
			want: true,
		},
		{
			name: "no converter is valid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.NoneConverter,
			}),
			want: true,
		},
		{
			name: "nil webhook converter is valid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.WebhookConverter,
				Webhook:  nil,
			}),
			want: true,
		},
		{
			name: "webhook converter with nil client config is valid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.WebhookConverter,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: nil,
				},
			}),
			want: true,
		},
		{
			name: "webhook converter with client config with nil ca bundle is invalid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.WebhookConverter,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: &apiextensionsv1.WebhookClientConfig{
						CABundle: nil,
					},
				},
			}),
			want: false,
		},
		{
			name: "equal data is valid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.WebhookConverter,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: &apiextensionsv1.WebhookClientConfig{
						CABundle: testValue1,
					},
				},
			}),
			want: true,
		},
		{
			name: "out of sync data is invalid",
			args: getCRDFromConversionSpec(&apiextensionsv1.CustomResourceConversion{
				Strategy: apiextensionsv1.WebhookConverter,
				Webhook: &apiextensionsv1.WebhookConversion{
					ClientConfig: &apiextensionsv1.WebhookClientConfig{
						CABundle: testValue2,
					},
				},
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, certSecret.isCRDConversionValid(tt.args), "isCRDConversionValid(%v)", tt.args)
		})
	}
}

func getCRDFromConversionSpec(conversionSpec *apiextensionsv1.CustomResourceConversion) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Conversion: conversionSpec,
		}}
}
