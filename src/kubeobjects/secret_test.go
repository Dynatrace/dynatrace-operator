package kubeobjects

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var secretLog = logger.Factory.GetLogger("test-secret")

func TestGetSecret(t *testing.T) {
	testSecretName := "testSecret"
	testNamespace := "testNamespace"
	fakeClient := fake.NewClient(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
		},
	)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, secretLog)

	t.Run("get existing secret", func(t *testing.T) {
		secret, err := secretQuery.Get(types.NamespacedName{Name: testSecretName, Namespace: testNamespace})

		require.NoError(t, err)
		assert.NotNil(t, secret)
	})
	t.Run("return error if secret does not exist", func(t *testing.T) {
		_, err := secretQuery.Get(types.NamespacedName{Name: "not a secret", Namespace: testNamespace})

		require.Error(t, err)
	})
}

func TestMultipleSecrets(t *testing.T) {
	testSecretName := "testSecret"
	// the query filter we use is not supported by the fake client => would return all secrets
	// therefore this list contains only one secret
	fakeClient := fake.NewClient(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns1",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns2",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns3",
			},
		},
	)
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, secretLog)

	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		secrets, err := secretQuery.GetAllFromNamespaces(testSecretName)
		require.NoError(t, err)
		assert.Len(t, secrets, 3)
	})
	t.Run("update and create secret in specific namespaces", func(t *testing.T) {
		namespaces := []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nsNotYetExisting",
				},
			},
		}

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns1",
			},
			Data: map[string][]byte{
				"samplekey": []byte("samplevalue"),
			},
		}
		err := secretQuery.CreateOrUpdateForNamespacesList(secret, namespaces)
		require.NoError(t, err)

		secrets, err := secretQuery.GetAllFromNamespaces(testSecretName)
		require.NoError(t, err)

		assert.Len(t, secrets, 4)
		secretsMap := make(map[string]corev1.Secret)
		for _, secret := range secrets {
			secretsMap[secret.Namespace] = secret
		}

		assert.Equal(t, secret.Data, secretsMap["ns1"].Data)
		assert.Equal(t, secret.Data, secretsMap["ns2"].Data)
		assert.Equal(t, secret.Data, secretsMap["nsNotYetExisting"].Data)

		assert.NotEqual(t, secret.Data, secretsMap["ns3"].Data)
	})
}

func TestInitialMultipleSecrets(t *testing.T) {
	testSecretName := "testSecret"
	fakeClient := fake.NewClient()
	secretQuery := NewSecretQuery(context.TODO(), fakeClient, fakeClient, secretLog)

	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		secrets, err := secretQuery.GetAllFromNamespaces(testSecretName)
		require.NoError(t, err)
		assert.Len(t, secrets, 0)
	})
}
