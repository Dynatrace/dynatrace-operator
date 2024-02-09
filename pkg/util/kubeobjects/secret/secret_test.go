package secret

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
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

var secretLog = logger.Get().WithName("test-secret")

const (
	testDeploymentName = "deployment-as-owner-of-secret"
	testSecretName     = "test-secret"
	testNamespace      = "test-namespace"
	testSecretDataKey  = "key"
)

var (
	dataValue  = []byte("dGVzdCB2YWx1ZSBudW1iZXIgMQ==")
	testSecret = corev1.Secret{
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
)

func createDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDeploymentName,
		},
	}
}

func TestGetSecret(t *testing.T) {
	fakeClient := fake.NewClient(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
		},
	)
	secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)

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

func TestMultipleSecrets(t *testing.T) {
	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)

		secrets, err := secretQuery.GetAllFromNamespaces(testSecretName)
		require.NoError(t, err)
		assert.Len(t, secrets, 3)
	})
	t.Run("update and create secret in specific namespaces", func(t *testing.T) {
		fakeClient := newClientWithSecrets()
		secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)

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
		err := secretQuery.CreateOrUpdateForNamespaces(secret, namespaces)
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
		secretQuery := NewQuery(context.Background(), boomClient, fakeReader, secretLog)

		err := secretQuery.CreateOrUpdateForNamespaces(secret, namespaces)
		require.Error(t, err)
		assert.NotEmpty(t, requestCounter)
	})
}

func TestInitialMultipleSecrets(t *testing.T) {
	testSecretName := "testSecret"
	fakeClient := fake.NewClientWithIndex()
	secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)

	t.Run("get existing secret from all namespaces", func(t *testing.T) {
		secrets, err := secretQuery.GetAllFromNamespaces(testSecretName)
		require.NoError(t, err)
		assert.Empty(t, secrets)
	})
}

func TestSecretBuilder(t *testing.T) {
	labelName := "name"
	labelValue := "value"
	labels := map[string]string{
		labelName: labelValue,
	}
	dataKey := ".dockercfg"
	dockerCfg := map[string][]byte{
		dataKey: {},
	}

	t.Run("create secret", func(t *testing.T) {
		secret, err := Create(scheme.Scheme, createDeployment(),
			NewNameModifier(testSecretName),
			NewNamespaceModifier(testNamespace))
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		assert.Empty(t, secret.Labels)
		assert.Equal(t, corev1.SecretType(""), secret.Type)
		assert.Empty(t, secret.Data)
	})
	t.Run("create secret with label", func(t *testing.T) {
		secret, err := Create(scheme.Scheme, createDeployment(),
			NewLabelsModifier(labels),
			NewNameModifier(testSecretName),
			NewNamespaceModifier(testNamespace),
			NewDataModifier(map[string][]byte{}))
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		require.Len(t, secret.Labels, 1)
		assert.Equal(t, labelValue, secret.Labels[labelName])
		assert.Equal(t, corev1.SecretType(""), secret.Type)
		assert.Empty(t, secret.Data)
	})
	t.Run("create secret with type", func(t *testing.T) {
		secret, err := Create(scheme.Scheme, createDeployment(),
			NewTypeModifier(corev1.SecretTypeDockercfg),
			NewNameModifier(testSecretName),
			NewNamespaceModifier(testNamespace),
			NewDataModifier(dockerCfg))
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		assert.Empty(t, secret.Labels)
		assert.Equal(t, corev1.SecretTypeDockercfg, secret.Type)
		_, found := secret.Data[dataKey]
		assert.True(t, found)
	})
	t.Run("create secret with label and type", func(t *testing.T) {
		secret, err := Create(scheme.Scheme, createDeployment(),
			NewLabelsModifier(labels),
			NewTypeModifier(corev1.SecretTypeDockercfg),
			NewNameModifier(testSecretName),
			NewNamespaceModifier(testNamespace),
			NewDataModifier(dockerCfg))
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		require.Len(t, secret.Labels, 1)
		assert.Equal(t, labelValue, secret.Labels[labelName])
		assert.Equal(t, corev1.SecretTypeDockercfg, secret.Type)
		_, found := secret.Data[dataKey]
		assert.True(t, found)
	})
}

func TestCreateOrUpdate(t *testing.T) {
	fakeClient := fake.NewClient()
	fakeClient.Create(context.Background(), testSecret.DeepCopy())

	t.Run("create secret", func(t *testing.T) {
		// empty client
		secretQuery := NewQuery(context.Background(), fake.NewClient(), fake.NewClient(), secretLog)

		err := secretQuery.CreateOrUpdate(testSecret)
		require.NoError(t, err)

		secret, _ := secretQuery.Get(types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.NotNil(t, secret)
	})
	t.Run("existing equal secret", func(t *testing.T) {
		// existing mocked secret in fakeClient
		secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)

		err := secretQuery.CreateOrUpdate(testSecret)
		require.NoError(t, err)

		secret, _ := secretQuery.Get(types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.NotNil(t, secret)
	})
	t.Run("update secret", func(t *testing.T) {
		// existing mocked secret in fakeClient
		secretQuery := NewQuery(context.Background(), fakeClient, fakeClient, secretLog)
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
		err := secretQuery.CreateOrUpdate(updatedSecret)
		require.NoError(t, err)

		secret, _ := secretQuery.Get(types.NamespacedName{Name: testSecretName, Namespace: testNamespace})
		assert.Equal(t, secret.Data[testSecretDataKey], newValue)
	})
}

func TestGetDataFromSecretName(t *testing.T) {
	fakeClient := fake.NewClient()
	fakeClient.Create(context.Background(), testSecret.DeepCopy())

	t.Run("get secret data", func(t *testing.T) {
		data, _ := GetDataFromSecretName(fakeClient, types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, testSecretDataKey, logger.DtLogger{})
		assert.Equal(t, string(dataValue), data)
	})
}
