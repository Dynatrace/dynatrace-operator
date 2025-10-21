package exporterconfig

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
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
	testCrt             = "test-cert"
	oldDataIngestToken  = "old-data-ingest-token"
	oldTestCert         = "old-test-cert"

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
		err := secretGenerator.GenerateForDynakube(t.Context(), dk)
		require.NoError(t, err)

		assertSecretNotFound(t, clt, GetSourceCertsSecretName(dk.Name), testNamespaceDynatrace)
		assertSecretNotFound(t, clt, consts.OTLPExporterCertsSecretName, testNamespace)
	})
	t.Run("successfully generate secrets for dynakube", func(t *testing.T) {
		tlsSecretName := "ag-tls-secret"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					Signals: otlpexporterconfiguration.SignalConfiguration{},
				},
				ActiveGate: activegate.Spec{
					TLSSecretName: tlsSecretName,
					Capabilities:  []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			clientSecret(tlsSecretName, testNamespaceDynatrace, map[string][]byte{
				dynakube.TLSCert: []byte(testCrt),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk)
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

		var sourceCertSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: GetSourceCertsSecretName(dk.Name), Namespace: dk.Namespace}, &sourceCertSecret)
		require.NoError(t, err)

		assert.Equal(t, testCrt, string(sourceCertSecret.Data[consts.ActiveGateCertDataName]))

		var certSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: consts.OTLPExporterCertsSecretName, Namespace: testNamespace}, &certSecret)
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data)

		assert.Equal(t, testCrt, string(certSecret.Data[consts.ActiveGateCertDataName]))

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionTrue, c.Status)
	})
	t.Run("update existing secrets", func(t *testing.T) {
		tlsSecretName := "ag-tls-secret"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					Signals: otlpexporterconfiguration.SignalConfiguration{},
				},
				ActiveGate: activegate.Spec{
					TLSSecretName: tlsSecretName,
					Capabilities:  []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			clientSecret(tlsSecretName, testNamespaceDynatrace, map[string][]byte{
				dynakube.TLSCert: []byte(testCrt),
			}),
			clientSecret(consts.OTLPExporterSecretName, testNamespace, map[string][]byte{
				dtclient.DataIngestToken: []byte(oldDataIngestToken),
			}),
			clientSecret(GetSourceConfigSecretName(dk.Name), dk.Namespace, map[string][]byte{
				dtclient.DataIngestToken: []byte(oldDataIngestToken),
			}),
			clientSecret(consts.OTLPExporterCertsSecretName, testNamespace, map[string][]byte{
				consts.ActiveGateCertDataName: []byte(oldTestCert),
			}),
			clientSecret(GetSourceCertsSecretName(dk.Name), dk.Namespace, map[string][]byte{
				consts.ActiveGateCertDataName: []byte(oldTestCert),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk)
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

		var sourceCertSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: GetSourceCertsSecretName(dk.Name), Namespace: dk.Namespace}, &sourceCertSecret)
		require.NoError(t, err)

		assert.Equal(t, testCrt, string(sourceCertSecret.Data[consts.ActiveGateCertDataName]))

		var certSecret corev1.Secret
		err = clt.Get(t.Context(), client.ObjectKey{Name: consts.OTLPExporterCertsSecretName, Namespace: testNamespace}, &certSecret)
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data)

		assert.Equal(t, testCrt, string(certSecret.Data[consts.ActiveGateCertDataName]))

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
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
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					Signals: otlpexporterconfiguration.SignalConfiguration{},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
		)

		mockDTClient := dtclientmock.NewClient(t)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(t.Context(), dk)
		require.Error(t, err)

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)
	})

	t.Run("generate secrets for multiple namespaces (skip terminating)", func(t *testing.T) {
		tlsSecretName := "ag-tls-secret"
		tlsCrt := []byte("dummycrt")

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{},
				ActiveGate: activegate.Spec{
					TLSSecretName: tlsSecretName,
					Capabilities:  []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName},
				},
			},
		}

		terminatingNS := clientInjectedNamespace("terminating-ns", testDynakube)
		terminatingNS.Status.Phase = corev1.NamespaceTerminating

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientInjectedNamespace(testNamespace2, testDynakube),
			terminatingNS,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{dtclient.DataIngestToken: []byte(testDataIngestToken)}),
			clientSecret(tlsSecretName, testNamespaceDynatrace, map[string][]byte{dynakube.TLSCert: tlsCrt}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.NoError(t, secretGenerator.GenerateForDynakube(t.Context(), dk))

		// replicated in active namespaces
		assertSecretExists(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretExists(t, clt, consts.OTLPExporterSecretName, testNamespace2)

		assertSecretExists(t, clt, consts.OTLPExporterCertsSecretName, testNamespace)
		assertSecretExists(t, clt, consts.OTLPExporterCertsSecretName, testNamespace2)
		// not replicated into terminating namespace
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, "terminating-ns")
		assertSecretNotFound(t, clt, consts.OTLPExporterCertsSecretName, "terminating-ns")
	})

	t.Run("token secret missing ingest token key -> return error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec:       dynakube.DynaKubeSpec{OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{}},
		}

		// tokens secret present but without ingest token key
		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{"other": []byte("value")}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.Error(t, secretGenerator.GenerateForDynakube(t.Context(), dk))

		c := meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType)
		require.NotNil(t, c)
		assert.Equal(t, metav1.ConditionFalse, c.Status)

		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
	})

	t.Run("no matching namespaces -> only source secret created", func(t *testing.T) {
		tlsSecretName := "ag-tls-secret"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testDynakube, Namespace: testNamespaceDynatrace},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{},
				ActiveGate: activegate.Spec{
					TLSSecretName: tlsSecretName,
					Capabilities:  []activegate.CapabilityDisplayName{activegate.MetricsIngestCapability.DisplayName},
				},
			},
		}

		// namespace without injection label
		nonInjected := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "plain-ns"}}

		clt := fake.NewClientWithIndex(
			dk,
			nonInjected,
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{dtclient.DataIngestToken: []byte(testDataIngestToken)}),
			clientSecret(tlsSecretName, testNamespaceDynatrace, map[string][]byte{dynakube.TLSCert: []byte(testCrt)}),
		)

		mockDTClient := dtclientmock.NewClient(t)
		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		require.NoError(t, secretGenerator.GenerateForDynakube(t.Context(), dk))

		// source secret exists
		assertSecretExists(t, clt, GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace)
		assertSecretExists(t, clt, GetSourceCertsSecretName(dk.Name), testNamespaceDynatrace)
		// replicated secret should not exist (no matching namespaces)
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, "plain-ns")
		assertSecretNotFound(t, clt, consts.OTLPExporterCertsSecretName, "plain-ns")
	})

	t.Run("missing tls secret referenced -> error", func(t *testing.T) {
		tlsSecretName := "missing-tls"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					TLSSecretName: tlsSecretName,
					Capabilities:  []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			dk,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{dtclient.DataIngestToken: []byte(testDataIngestToken)}),
		)

		sg := NewSecretGenerator(clt, clt, dtclientmock.NewClient(t))

		require.Error(t, sg.GenerateForDynakube(t.Context(), dk))

		assertSecretNotFound(t, clt, GetSourceCertsSecretName(dk.Name), testNamespaceDynatrace)
		assertSecretNotFound(t, clt, consts.OTLPExporterCertsSecretName, testNamespace)
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
				{Type: ConfigConditionType},
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

	assertSecretNotFound(t, clt, GetSourceCertsSecretName(dk.Name), testNamespaceDynatrace)

	require.Nil(t, meta.FindStatusCondition(*dk.Conditions(), ConfigConditionType))
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
