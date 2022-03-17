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
)

const (
	testValue1 = "test-value"
	testValue2 = "test-alternative-value"
	testKey1   = "test-key"
	testKey2   = "test-name"

	testSecretName = "super-secret"
	testSecretType = corev1.SecretType("type")
)

var log = logger.NewDTLogger()

func TestCreateOrUpdateSecretIfNotExists(t *testing.T) {
	t.Run(`Secret present, no change`, func(t *testing.T) {
		data := map[string][]byte{testKey1: []byte(testValue1)}
		labels := map[string]string{
			"label": "test",
		}
		client := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
				Labels:    labels,
			},
			Data: data,
		})

		_, err := CreateOrUpdateSecretIfNotExists(client, client, testSecretName, testNamespace, data, labels, "", log)
		assert.NoError(t, err)
	})
	t.Run(`Secret present, different data => update data`, func(t *testing.T) {
		data := map[string][]byte{testKey1: []byte(testValue1)}
		labels := map[string]string{
			"label": "test",
		}
		client := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
				Labels:    labels,
			},
			Data: map[string][]byte{},
		})

		_, err := CreateOrUpdateSecretIfNotExists(client, client, testSecretName, testNamespace, data, labels, testSecretType, log)
		assert.NoError(t, err)

		var updatedSecret corev1.Secret
		client.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &updatedSecret)
		assert.True(t, reflect.DeepEqual(data, updatedSecret.Data))
	})
	t.Run(`Secret present, different labels => update labels`, func(t *testing.T) {
		data := map[string][]byte{testKey1: []byte(testValue1)}
		labels := map[string]string{
			"label": "test",
		}
		client := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
				Labels:    map[string]string{},
			},
			Data: data,
		})

		_, err := CreateOrUpdateSecretIfNotExists(client, client, testSecretName, testNamespace, data, labels, testSecretType, log)
		assert.NoError(t, err)

		var updatedSecret corev1.Secret
		client.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &updatedSecret)
		assert.True(t, reflect.DeepEqual(labels, updatedSecret.Labels))
	})
	t.Run(`Secret in other namespace => create in targe namespace`, func(t *testing.T) {
		data := map[string][]byte{testKey1: []byte(testValue1)}
		labels := map[string]string{
			"label": "test",
		}
		client := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "other",
			},
			Data: map[string][]byte{},
		})

		_, err := CreateOrUpdateSecretIfNotExists(client, client, testSecretName, testNamespace, data, labels, testSecretType, log)
		assert.NoError(t, err)

		var newSecret corev1.Secret
		client.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &newSecret)
		assert.True(t, reflect.DeepEqual(data, newSecret.Data))
		assert.True(t, reflect.DeepEqual(labels, newSecret.Labels))
		assert.Equal(t, testSecretName, newSecret.Name)
		assert.Equal(t, testNamespace, newSecret.Namespace)
		assert.Equal(t, testSecretType, newSecret.Type)
	})
	t.Run(`Secret not present => create in targe namespace`, func(t *testing.T) {
		data := map[string][]byte{testKey1: []byte(testValue1)}
		labels := map[string]string{
			"label": "test",
		}
		client := fake.NewClient()

		_, err := CreateOrUpdateSecretIfNotExists(client, client, testSecretName, testNamespace, data, labels, testSecretType, log)
		assert.NoError(t, err)

		var newSecret corev1.Secret
		client.Get(context.TODO(), types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, &newSecret)
		assert.True(t, reflect.DeepEqual(data, newSecret.Data))
		assert.True(t, reflect.DeepEqual(labels, newSecret.Labels))
		assert.Equal(t, testSecretName, newSecret.Name)
		assert.Equal(t, testNamespace, newSecret.Namespace)
		assert.Equal(t, testSecretType, newSecret.Type)
	})
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
