package k8ssecret

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var secretLog = logd.Get().WithName("test-secret")

const (
	testDeploymentName = "deployment-as-owner-of-secret"
	testSecretName     = "test-secret"
	testNamespace      = "test-namespace"
	testSecretDataKey  = "key"
)

var (
	dataValue = []byte("dGVzdCB2YWx1ZSBudW1iZXIgMQ==")
)

func getTestSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			testSecretDataKey: dataValue,
		},
	}
}

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClient(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
		},
	)
	secretQuery := Query(fakeClient, fakeClient, secretLog)

	t.Run("get existing secret", func(t *testing.T) {
		secret, err := secretQuery.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespace})

		require.NoError(t, err)
		assert.NotNil(t, secret)
	})
	t.Run("return error if secret does not exist", func(t *testing.T) {
		_, err := secretQuery.Get(ctx, types.NamespacedName{Name: "not a secret", Namespace: testNamespace})

		require.Error(t, err)
	})
}

func newClientWithSecrets() client.Client {
	return fake.NewClientWithIndex(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns1",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
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
				Name:      "other",
				Namespace: "ns2",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: "ns3",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: "ns3",
			},
		},
	)
}

func TestMultipleNamespaces(t *testing.T) {
	ctx := context.Background()

	t.Run("deletion of test secret in namespaces 1 and 2", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := Query(fakeClient, fakeClient, secretLog)

		namespaces := []string{"ns1", "ns2"}
		err := secretQuery.DeleteForNamespaces(ctx, testSecretName, namespaces)
		require.NoError(t, err)

		// get all secrets from all namespaces
		secretList := &corev1.SecretList{}
		err = fakeClient.List(context.Background(), secretList)

		require.NoError(t, err)
		assert.Len(t, secretList.Items, 4)
	})
	t.Run("deletion of test secret in namespaces 1 and 2 and empty", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := Query(fakeClient, fakeClient, secretLog)

		// secret does not exist in this namespace => other secrets should still get deleted
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "empty",
			},
		}
		_ = fakeClient.Create(context.Background(), &ns)

		namespaces := []string{"ns1", "ns2", "empty"}
		err := secretQuery.DeleteForNamespaces(ctx, testSecretName, namespaces)
		require.NoError(t, err)

		// get all secrets from all namespaces
		secretList := &corev1.SecretList{}
		err = fakeClient.List(context.Background(), secretList)

		require.NoError(t, err)
		assert.Len(t, secretList.Items, 4)
	})
}

func TestMultipleSecrets(t *testing.T) {
	ctx := context.Background()

	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := Query(fakeClient, fakeClient, secretLog)

		secrets, err := secretQuery.GetAllFromNamespaces(ctx, testSecretName)
		require.NoError(t, err)
		assert.Len(t, secrets, 3)
	})
	t.Run("update and create secret in specific namespaces", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := Query(fakeClient, fakeClient, secretLog)

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
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nsTerminating",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
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
		err := secretQuery.CreateOrUpdateForNamespaces(ctx, &secret, namespaces)
		require.NoError(t, err)

		secrets, err := secretQuery.GetAllFromNamespaces(ctx, testSecretName)
		require.NoError(t, err)

		assert.Len(t, secrets, 4)

		secretsMap := make(map[string]corev1.Secret)
		for _, secret := range secrets {
			secretsMap[secret.Namespace] = *secret
		}

		assert.Equal(t, secret.Data, secretsMap["ns1"].Data)
		assert.Equal(t, secret.Data, secretsMap["ns2"].Data)
		assert.Equal(t, secret.Data, secretsMap["nsNotYetExisting"].Data)

		assert.NotEqual(t, secret.Data, secretsMap["ns3"].Data)
	})
	t.Run("only 1 error because of kubernetes rejecting the request", func(t *testing.T) {
		requestCounter := 0
		fakeReader := newClientWithSecrets()
		boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				requestCounter++

				return errors.New("BOOM")
			},
			Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				requestCounter++

				return errors.New("BOOM")
			},
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				requestCounter++

				return errors.New("BOOM")
			},
		})
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: testSecretName,
			},
			Data: map[string][]byte{
				"samplekey": []byte("samplevalue"),
			},
		}
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
		secretQuery := Query(boomClient, fakeReader, secretLog)

		err := secretQuery.CreateOrUpdateForNamespaces(ctx, &secret, namespaces)
		require.Error(t, err)
		assert.NotEmpty(t, requestCounter)
	})
}

func TestInitialMultipleSecrets(t *testing.T) {
	testSecretName := "testSecret"
	fakeClient := fake.NewClientWithIndex()
	secretQuery := Query(fakeClient, fakeClient, secretLog)

	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		secrets, err := secretQuery.GetAllFromNamespaces(context.Background(), testSecretName)
		require.NoError(t, err)
		assert.Empty(t, secrets)
	})
}

func TestCreateOrUpdate(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClient()
	fakeClient.Create(context.Background(), getTestSecret())

	t.Run("create secret", func(t *testing.T) {
		// empty client
		secretQuery := Query(fake.NewClient(), fake.NewClient(), secretLog)

		created, err := secretQuery.CreateOrUpdate(ctx, getTestSecret())
		require.NoError(t, err)
		require.True(t, created)

		secret, _ := secretQuery.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.NotNil(t, secret)
	})
	t.Run("existing equal secret", func(t *testing.T) {
		// existing mocked secret in fakeClient
		secretQuery := Query(fakeClient, fakeClient, secretLog)

		updated, err := secretQuery.CreateOrUpdate(ctx, getTestSecret())
		require.NoError(t, err)
		require.False(t, updated)

		secret, _ := secretQuery.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.NotNil(t, secret)
	})
	t.Run("update secret", func(t *testing.T) {
		// existing mocked secret in fakeClient
		secretQuery := Query(fakeClient, fakeClient, secretLog)
		newValue := []byte("dGVzdCB2YWx1ZSBudW1iZXIgMg==")
		updatedSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				testSecretDataKey: newValue,
			},
		}
		updated, err := secretQuery.CreateOrUpdate(ctx, &updatedSecret)
		require.NoError(t, err)
		require.True(t, updated)

		secret, _ := secretQuery.Get(ctx, types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.Equal(t, secret.Data[testSecretDataKey], newValue)
	})
}
