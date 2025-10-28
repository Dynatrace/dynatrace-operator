package exporterconfig

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDataIngestToken = "test-ingest-token"
	oldDataIngestToken  = "old-data-ingest-token"

	testDynakube   = "test-dynakube"
	testNamespace  = "test-namespace"
	testNamespace2 = "test-namespace2"

	testNamespaceDynatrace = "dynatrace"
)

func TestNewSecretGenerator(t *testing.T) {
	client := fake.NewClient()
	mockDTClient := dtclientmock.NewClient(t)

	secretGenerator := NewSecretGenerator(client, client, mockDTClient)
	assert.NotNil(t, secretGenerator)

	assert.Equal(t, client, secretGenerator.client)
	assert.Equal(t, client, secretGenerator.apiReader)
	assert.Equal(t, mockDTClient, secretGenerator.dtClient)
}

func TestSecretGenerator_GenerateForDynakube(t *testing.T) {
	t.Run("no OTLP exporter config enabled - do not create secret", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk, nil)
		require.NoError(t, err)

		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretNotFound(t, clt, GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace)
	})
	t.Run("no namespaces provided - should not error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.Spec{
					Signals: otlp.SignalConfiguration{},
				},
			},
		}

		namespace := clientInjectedNamespace(testNamespace, testDynakube)

		clt := fake.NewClientWithIndex(
			dk,
			namespace,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk, nil)
		require.NoError(t, err)
	})
	t.Run("successfully generate config secret for dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.Spec{
					Signals: otlp.SignalConfiguration{},
				},
			},
		}

		namespace := clientInjectedNamespace(testNamespace, testDynakube)

		clt := fake.NewClientWithIndex(
			dk,
			namespace,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk, []corev1.Namespace{*namespace})
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: consts.OTLPExporterSecretName, Namespace: testNamespace}, &secret)
		require.NoError(t, err)
		require.Equal(t, consts.OTLPExporterSecretName, secret.Name)
		assert.NotEmpty(t, secret.Data)

		assert.Equal(t, testDataIngestToken, string(secret.Data[dtclient.DataIngestToken]))

		var sourceSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &sourceSecret)
		require.NoError(t, err)

		require.Equal(t, GetSourceConfigSecretName(dk.Name), sourceSecret.Name)
		assert.Equal(t, secret.Data, sourceSecret.Data)

		c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("update existing secret", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.Spec{
					Signals: otlp.SignalConfiguration{},
				},
			},
		}

		namespace := clientInjectedNamespace(testNamespace, testDynakube)

		clt := fake.NewClientWithIndex(
			dk,
			namespace,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			clientSecret(consts.OTLPExporterSecretName, testNamespace, map[string][]byte{
				dtclient.DataIngestToken: []byte(oldDataIngestToken),
			}),
			clientSecret(GetSourceConfigSecretName(dk.Name), dk.Namespace, map[string][]byte{
				dtclient.DataIngestToken: []byte(oldDataIngestToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk, []corev1.Namespace{*namespace})
		require.NoError(t, err)

		var secret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: consts.OTLPExporterSecretName, Namespace: testNamespace}, &secret)
		require.NoError(t, err)
		require.Equal(t, consts.OTLPExporterSecretName, secret.Name)
		assert.NotEmpty(t, secret.Data)

		assert.Equal(t, testDataIngestToken, string(secret.Data[dtclient.DataIngestToken]))

		var sourceSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: GetSourceConfigSecretName(dk.Name), Namespace: dk.Namespace}, &sourceSecret)
		require.NoError(t, err)

		require.Equal(t, GetSourceConfigSecretName(dk.Name), sourceSecret.Name)
		assert.Equal(t, secret.Data, sourceSecret.Data)

		c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("fail while generating secret for dynakube - secret in dynatrace namespace not found", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlp.Spec{
					Signals: otlp.SignalConfiguration{},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk, nil)
		require.Error(t, err)

		c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)
	})

	t.Run("generate secrets for multiple namespaces (skip terminating)", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec:       dynakube.DynaKubeSpec{OTLPExporterConfiguration: &otlp.Spec{}},
		}

		terminatingNS := clientInjectedNamespace("terminating-ns", testDynakube)
		terminatingNS.Status.Phase = corev1.NamespaceTerminating

		namespace1 := clientInjectedNamespace(testNamespace, testDynakube)
		namespace2 := clientInjectedNamespace(testNamespace2, testDynakube)

		clt := fake.NewClientWithIndex(
			dk,
			namespace1,
			namespace2,
			terminatingNS,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{dtclient.DataIngestToken: []byte(testDataIngestToken)}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.NoError(t, secretGenerator.GenerateForDynakube(t.Context(), dk, []corev1.Namespace{*namespace1, *namespace2, *terminatingNS}))

		// replicated in active namespaces
		assertSecretExists(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretExists(t, clt, consts.OTLPExporterSecretName, testNamespace2)
		// not replicated into terminating namespace
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, "terminating-ns")
	})

	t.Run("token secret missing ingest token key -> return error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec:       dynakube.DynaKubeSpec{OTLPExporterConfiguration: &otlp.Spec{}},
		}

		// tokens secret present but without ingest token key
		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{"other": []byte("value")}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.Error(t, secretGenerator.GenerateForDynakube(t.Context(), dk, nil))

		c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)

		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
	})

	t.Run("no matching namespaces -> only source secret created", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec:       dynakube.DynaKubeSpec{OTLPExporterConfiguration: &otlp.Spec{}},
		}

		// namespace without injection label
		nonInjected := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "plain-ns"}}

		clt := fake.NewClientWithIndex(
			dk,
			nonInjected,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{dtclient.DataIngestToken: []byte(testDataIngestToken)}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.NoError(t, secretGenerator.GenerateForDynakube(t.Context(), dk, nil))

		// source secret exists
		assertSecretExists(t, clt, GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace)
		// replicated secret should not exist (no matching namespaces)
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, "plain-ns")
	})
}

func TestCleanup(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynakube.DynaKubeSpec{},
		Status: dynakube.DynaKubeStatus{
			Conditions: []metav1.Condition{
				{Type: ConditionType},
				{Type: "other"},
			},
		},
	}

	clt := fake.NewClientWithIndex(
		dk,
		clientInjectedNamespace(testNamespace, testDynakube),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			dtclient.DataIngestToken: []byte(testDataIngestToken),
		}),
		clientSecret(consts.OTLPExporterSecretName, testNamespace, nil),
		clientSecret(consts.OTLPExporterSecretName, testNamespace2, nil),
		clientSecret(GetSourceConfigSecretName(dk.Name), dk.Namespace, nil),
	)
	namespaces := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace2}},
	}

	err := Cleanup(t.Context(), clt, clt, namespaces, dk)
	require.NoError(t, err)

	assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
	assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace2)
	assertSecretNotFound(t, clt, GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace)

	c := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
	assert.Nil(t, c)
}

func clientSecret(secretName string, namespaceName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		Data: data,
	}
}

func clientInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: dynakubeName,
			},
		},
	}
}

func assertSecretExists(t *testing.T, clt client.Client, name, namespace string) {
	var s corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: namespace}, &s)
	require.NoError(t, err)
}

func assertSecretNotFound(t *testing.T, clt client.Client, name, namespace string) {
	var s corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: namespace}, &s)
	assert.True(t, errors.IsNotFound(err), "expected not found, got %v", err)
}
