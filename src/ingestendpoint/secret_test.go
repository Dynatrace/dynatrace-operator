package ingestendpoint

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	paasToken              = "test-paas-token"
	apiToken               = "test-api-token"
	dataIngestToken        = "test-data-ingest-token"
	updatedDataIngestToken = "updated-test-data-ingest-token"

	apiUrl        = "https://tenant.test/api"
	updatedApiUrl = "https://tenant.updated-test/api"

	dataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	updatedTokenDataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=updated-test-data-ingest-token
`
	updatedApiUrlDataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.updated-test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	dataIngestSecretLocalAGWithMetrics = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	updatedApiUrlDataIngestSecretLocalAgWithMetrics = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	dataIngestSecretLocalAGWithMetricsAndStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`

	updatedApiUrlDataIngestSecretLocalAGWithStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`
	emptyFile = ``

	namespace1 = "test-namespace-one"
	namespace2 = "test-namespace-two"

	namespaceDynatrace = "dynatrace"
	dynakubeName       = "dynakube"
)

func TestGenerateDataIngestSecret_ForDynakube(t *testing.T) {
	t.Run(`data-ingest endpoint secret created but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, namespaceDynatrace)

		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, false, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`data-ingest endpoint secret created and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, namespaceDynatrace)

		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}

		updateTestSecret(t, fakeClient)

		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedTokenDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`data-ingest endpoint secret created and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, namespaceDynatrace)

		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}

		updateTestDynakube(t, fakeClient)

		{
			upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), dynakubeName, namespace1)
			assert.NoError(t, err)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})

	t.Run(`data-ingest endpoint secret created in all namespaces but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, false, upd)
		}
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
		}

		updateTestSecret(t, fakeClient)

		{
			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedTokenDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, updatedTokenDataIngestSecretWithMetrics)
		}

		checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()

			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			newInstance := updatedTestDynakube()

			upd := testGenerateEndpointsSecret(t, newInstance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`data-ingest endpoint secret created (local AG) in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())
		{
			instance := buildTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, newInstance, fakeClient)
			assert.Equal(t, false, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAgWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAgWithMetrics)

			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`metrics-ingest with statsd endpoint secret created (local AG) in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetricsAndStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetricsAndStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, newInstance, fakeClient)
			assert.Equal(t, false, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`StatsD ingest URL is added/removed to endpoint properties when statsd-ingest capability is added/removed`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}

		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, newInstance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, updatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			newerInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, newerInstance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
		{
			unchangedInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			upd := testGenerateEndpointsSecret(t, unchangedInstance, fakeClient)
			assert.Equal(t, false, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, dataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
	t.Run(`No ingestion is enabled (statsd capability is not enabled, disable-metadata-enrichment feature flag is set true)`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()
			instance.Annotations = map[string]string{
				dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment: "true",
			}

			upd := testGenerateEndpointsSecret(t, instance, fakeClient)
			assert.Equal(t, true, upd)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName}, emptyFile)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName}, emptyFile)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespaceDynatrace, Name: SecretEndpointName})
		}
	})
}

func testGenerateEndpointsSecret(t *testing.T, instance *dynatracev1beta1.DynaKube, fakeClient client.Client) bool {
	endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, namespaceDynatrace)

	upd, err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
	assert.NoError(t, err)
	return upd
}

func TestRemoveEndpointSecrets(t *testing.T) {
	dk := buildTestDynakube()
	fakeClient := buildTestClientAfterGenerate(dk)

	endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, dk.Namespace)

	err := endpointSecretGenerator.RemoveEndpointSecrets(context.TODO(), dk)
	require.NoError(t, err)

	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace1, Name: SecretEndpointName})
	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: namespace2, Name: SecretEndpointName})

}

func checkTestSecretContains(t *testing.T, fakeClient client.Client, secretName types.NamespacedName, data string) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), secretName, &testSecret)
	assert.NoError(t, err)
	assert.NotNil(t, testSecret.Data)
	assert.NotEmpty(t, testSecret.Data)
	assert.Contains(t, testSecret.Data, "endpoint.properties")
	assert.Equal(t, data, string(testSecret.Data["endpoint.properties"]))
}

func checkTestSecretDoesntExist(t *testing.T, fakeClient client.Client, secretName types.NamespacedName) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), secretName, &testSecret)
	assert.Error(t, err)
	assert.Nil(t, testSecret.Data)
}

func updateTestSecret(t *testing.T, fakeClient client.Client) {
	updatedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: namespaceDynatrace,
		},
		Data: map[string][]byte{
			"apiToken":        []byte(apiToken),
			"paasToken":       []byte(paasToken),
			"dataIngestToken": []byte(updatedDataIngestToken),
		},
	}

	err := fakeClient.Update(context.TODO(), updatedSecret)
	assert.NoError(t, err)
}

func updatedTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: namespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: updatedApiUrl,
		},
	}
}

func updatedTestDynakubeWithDataIngestCapability(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: namespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL: updatedApiUrl,
		},
	}
}

func updateTestDynakube(t *testing.T, fakeClient client.Client) {
	var dk dynatracev1beta1.DynaKube
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakubeName, Namespace: namespaceDynatrace}, &dk)
	assert.NoError(t, err)

	dk.Spec.APIURL = updatedApiUrl

	err = fakeClient.Update(context.TODO(), &dk)
	assert.NoError(t, err)
}

func buildTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: namespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: apiUrl,
		},
	}
}

func buildTestDynakubeWithDataIngestCapability(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: namespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL: apiUrl,
		},
	}
}

func buildTestClientBeforeGenerate(dk *dynatracev1beta1.DynaKube) client.Client {
	return fake.NewClient(dk,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceDynatrace,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace1,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace2,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: namespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(apiToken),
				"paasToken":       []byte(paasToken),
				"dataIngestToken": []byte(dataIngestToken),
			},
		})
}

func buildTestClientAfterGenerate(dk *dynatracev1beta1.DynaKube) client.Client {
	return fake.NewClient(dk,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceDynatrace,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace1,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace2,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespace1,
				Namespace: namespaceDynatrace,
			},
			Data: map[string][]byte{
				"doesn't": []byte("matter"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespace2,
				Namespace: namespaceDynatrace,
			},
			Data: map[string][]byte{
				"doesn't": []byte("matter"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: namespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(apiToken),
				"paasToken":       []byte(paasToken),
				"dataIngestToken": []byte(dataIngestToken),
			},
		})
}
