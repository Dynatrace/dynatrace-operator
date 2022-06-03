package kubeobjects

import (
	"context"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logger.NewDTLogger()

func TestSecretQuery(t *testing.T) {
	t.Run(`Get secret`, testGetSecret)
	t.Run(`Create secret`, testCreateSecret)
	t.Run(`Update secret`, testUpdateSecret)
	t.Run(`Create or update secret`, testCreateOrUpdateSecret)
	t.Run(`Identical secret is not updated`, testIdenticalSecretIsNotUpdated)
	t.Run(`Update secret when data has changed`, testUpdateSecretWhenDataChanged)
	t.Run(`Update secret when labels have changed`, testUpdateSecretWhenLabelsChanged)
	t.Run(`Create secret in target namespace`, testCreateSecretInTargetNamespace)
}

func testGetSecret(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{testKey1: []byte(testSecretValue)},
	}
	fakeClient := fake.NewClient(&secret)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	foundSecret, err := secretQuery.Get(client.ObjectKey{Name: testSecretName, Namespace: testNamespace})

	assert.NoError(t, err)
	assert.True(t, AreSecretsEqual(secret, foundSecret))
}

func testCreateSecret(t *testing.T) {
	fakeClient := fake.NewClient()
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{testKey1: []byte(testSecretValue)},
	}

	err := secretQuery.Create(secret)

	assert.NoError(t, err)

	var actualSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testSecretName, Namespace: testNamespace}, &actualSecret)

	assert.NoError(t, err)
	assert.True(t, AreSecretsEqual(secret, actualSecret))
}

func testUpdateSecret(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{testKey1: []byte(testSecretValue)},
	}
	fakeClient := fake.NewClient()
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.Update(secret)

	assert.Error(t, err)

	secret.Data = nil
	fakeClient = fake.NewClient(&secret)
	secretQuery.kubeClient = fakeClient

	err = secretQuery.Update(secret)

	assert.NoError(t, err)

	var updatedSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, &updatedSecret)

	assert.NoError(t, err)
	assert.True(t, AreSecretsEqual(secret, updatedSecret))
}

func testCreateOrUpdateSecret(t *testing.T) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{testKey1: []byte(testSecretValue)},
	}
	fakeClient := fake.NewClient()
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.CreateOrUpdate(secret)

	assert.NoError(t, err)

	var createdSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, &createdSecret)

	assert.NoError(t, err)
	assert.True(t, AreSecretsEqual(secret, createdSecret))

	fakeClient = fake.NewClient(&secret)
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: nil,
	}
	secretQuery.kubeClient = fakeClient

	err = secretQuery.CreateOrUpdate(secret)

	assert.NoError(t, err)

	var updatedSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: secret.Name, Namespace: secret.Namespace}, &updatedSecret)

	assert.NoError(t, err)
	assert.True(t, AreSecretsEqual(secret, updatedSecret))
}

func testIdenticalSecretIsNotUpdated(t *testing.T) {
	data := map[string][]byte{testKey1: []byte(testValue1)}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: data,
	})
	secret := createTestSecret(labels, data)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.CreateOrUpdate(*secret)
	assert.NoError(t, err)
}

func testUpdateSecretWhenDataChanged(t *testing.T) {
	data := map[string][]byte{testKey1: []byte(testValue1)}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: map[string][]byte{},
	})
	secret := createTestSecret(labels, data)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.CreateOrUpdate(*secret)
	assert.NoError(t, err)

	var updatedSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &updatedSecret)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, updatedSecret.Data))
}

func testUpdateSecretWhenLabelsChanged(t *testing.T) {
	data := map[string][]byte{testKey1: []byte(testValue1)}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
			Labels:    map[string]string{},
		},
		Data: data,
	})
	secret := createTestSecret(labels, data)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.CreateOrUpdate(*secret)
	assert.NoError(t, err)

	var updatedSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &updatedSecret)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(labels, updatedSecret.Labels))
}

func testCreateSecretInTargetNamespace(t *testing.T) {
	data := map[string][]byte{testKey1: []byte(testValue1)}
	labels := map[string]string{
		"label": "test",
	}
	fakeClient := fake.NewClient(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: "other",
		},
		Data: map[string][]byte{},
	})
	secret := createTestSecret(labels, data)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, log)

	err := secretQuery.CreateOrUpdate(*secret)

	assert.NoError(t, err)

	var newSecret corev1.Secret
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &newSecret)

	assert.NoError(t, err)
	assert.True(t, reflect.DeepEqual(data, newSecret.Data))
	assert.True(t, reflect.DeepEqual(labels, newSecret.Labels))
	assert.Equal(t, testSecretName, newSecret.Name)
	assert.Equal(t, testNamespace, newSecret.Namespace)
	assert.Equal(t, corev1.SecretTypeOpaque, newSecret.Type)
}

func TestNewTokens(t *testing.T) {
	t.Run(`NewTokens extracts api and paas token from secret`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatraceApiToken:  []byte(testValue1),
				dtclient.DynatracePaasToken: []byte(testValue2),
			}}
		tokens, err := NewTokens(&secret)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, testValue1, tokens.ApiToken)
		assert.Equal(t, testValue2, tokens.PaasToken)
	})
	t.Run(`NewTokens handles missing api or paas token`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatraceApiToken: []byte(testValue1),
			}}
		tokens, err := NewTokens(&secret)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, testValue1, tokens.ApiToken)
		assert.Equal(t, testValue1, tokens.PaasToken)

		secret = corev1.Secret{
			Data: map[string][]byte{
				dtclient.DynatracePaasToken: []byte(testValue2),
			}}
		tokens, err = NewTokens(&secret)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), dtclient.DynatraceApiToken)

		secret = corev1.Secret{
			Data: map[string][]byte{}}
		tokens, err = NewTokens(&secret)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), dtclient.DynatraceApiToken)
	})
	t.Run(`NewTokens handles nil secret`, func(t *testing.T) {
		tokens, err := NewTokens(nil)

		assert.Error(t, err)
		assert.Nil(t, tokens)
	})
}

func TestExtractToken(t *testing.T) {
	t.Run(`ExtractToken returns value from secret`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{
				testKey1: []byte(testValue1),
				testKey2: []byte(testValue2),
			}}

		value, err := ExtractToken(&secret, testKey1)

		assert.NoError(t, err)
		assert.Equal(t, value, testValue1)

		value, err = ExtractToken(&secret, testKey2)

		assert.NoError(t, err)
		assert.Equal(t, value, testValue2)
	})
	t.Run(`ExtractToken handles missing key`, func(t *testing.T) {
		secret := corev1.Secret{
			Data: map[string][]byte{}}

		value, err := ExtractToken(&secret, testKey1)

		assert.Error(t, err)
		assert.Empty(t, value)
	})
}

func createTestSecret(labels map[string]string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	return secret
}
