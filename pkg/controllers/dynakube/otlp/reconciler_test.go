package otlp

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/otlpexporterconfiguration"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otlp/exporterconfig"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDataIngestToken = "test-ingest-token"

	testDynakube   = "test-dynakube"
	testDynakube2  = "test-dynakube2"
	testNamespace  = "test-namespace"
	testNamespace2 = "test-namespace2"

	testNamespaceSelectorLabel = "namespaceSelector"

	testNamespaceDynatrace = "dynatrace"
)

// failingReader simulates failures on List operations
type failingReader struct {
	client.Reader
	fail bool
}

func (f failingReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error { //nolint:revive
	if f.fail {
		return errors.New("err")
	}
	return f.Reader.List(ctx, list, opts...)
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("reconcile OTLP exporter configuration", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					NamespaceSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							testNamespaceSelectorLabel: testDynakube,
						},
					},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			clientInjectedNamespace(testNamespace, testDynakube),
			clientNotInjectedNamespace(testNamespace2, testDynakube2),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			dk,
		)
		dtClient := dtclientmock.NewClient(t)

		rec := NewReconciler(clt, clt, dtClient, dk)

		err := rec.Reconcile(t.Context())
		require.NoError(t, err)

		assertSecretFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace2)

		assert.NotNil(t, meta.FindStatusCondition(*dk.Conditions(), otlpExporterConfigurationConditionType))
	})

	t.Run("no exporter config triggers cleanup", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{},
				},
			},
		}

		setOTLPExporterConfigurationCondition(dk.Conditions())

		// pre-create a replicated secret and source secret to ensure cleanup removes them
		clt := fake.NewClientWithIndex(
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(consts.OTLPExporterSecretName, testNamespace, map[string][]byte{"foo": []byte("bar")}),
			clientSecret(exporterconfig.GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace, map[string][]byte{"foo": []byte("bar")}),
			dk,
		)

		dtClient := dtclientmock.NewClient(t)
		rec := NewReconciler(clt, clt, dtClient, dk)

		err := rec.Reconcile(t.Context())
		require.NoError(t, err)

		// secrets should be gone after cleanup
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)
		assertSecretNotFound(t, clt, exporterconfig.GetSourceConfigSecretName(dk.Name), testNamespaceDynatrace)

		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), otlpExporterConfigurationConditionType))
	})

	t.Run("missing tokens secret returns error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					NamespaceSelector: metav1.LabelSelector{MatchLabels: map[string]string{testNamespaceSelectorLabel: testDynakube}},
				},
			},
		}

		clt := fake.NewClientWithIndex(
			clientInjectedNamespace(testNamespace, testDynakube),
			dk,
		)

		dtClient := dtclientmock.NewClient(t)
		rec := NewReconciler(clt, clt, dtClient, dk)

		err := rec.Reconcile(t.Context())
		require.Error(t, err, "expected error due to missing tokens secret")
		assertSecretNotFound(t, clt, consts.OTLPExporterSecretName, testNamespace)

		found := false
		for _, cond := range dk.Status.Conditions {
			if cond.Type == otlpExporterConfigurationConditionType {
				found = true
				break
			}
		}
		assert.True(t, found, "expected OTLPExporterConfiguration condition to be set")
	})

	t.Run("mapper list namespaces error returns error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				OTLPExporterConfiguration: &otlpexporterconfiguration.Spec{
					NamespaceSelector: metav1.LabelSelector{MatchLabels: map[string]string{testNamespaceSelectorLabel: testDynakube}},
				},
			},
		}

		baseClient := fake.NewClientWithIndex(
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.DataIngestToken: []byte(testDataIngestToken),
			}),
			dk,
		)

		fReader := failingReader{Reader: baseClient, fail: true}

		dtClient := dtclientmock.NewClient(t)
		rec := NewReconciler(baseClient, fReader, dtClient, dk)

		err := rec.Reconcile(t.Context())
		require.Error(t, err)
	})
}

func assertSecretFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.NoError(t, err, "%s.%s secret not found, error: %s", secretName, secretNamespace, err)
}

func assertSecretNotFound(t *testing.T, clt client.Client, secretName string, secretNamespace string) {
	var secret corev1.Secret
	err := clt.Get(t.Context(), client.ObjectKey{Name: secretName, Namespace: secretNamespace}, &secret)
	require.Error(t, err, "%s.%s secret found, error: %s ", secretName, secretNamespace, err)
	assert.True(t, k8serrors.IsNotFound(err), "%s.%s secret, unexpected error: %s", secretName, secretNamespace, err)
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
				testNamespaceSelectorLabel:       dynakubeName,
			},
		},
	}
}

func clientNotInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				testNamespaceSelectorLabel: dynakubeName,
			},
		},
	}
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
