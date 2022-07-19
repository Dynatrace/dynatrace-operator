package ingestendpoint

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken              = "test-paas-token"
	testApiToken               = "test-api-token"
	testDataIngestToken        = "test-data-ingest-token"
	testUpdatedDataIngestToken = "updated-test-data-ingest-token"

	testApiUrl        = "https://tenant.test/api"
	testUpdatedApiUrl = "https://tenant.updated-test/api"

	testDataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedTokenDataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=updated-test-data-ingest-token
`
	testUpdatedApiUrlDataIngestSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.updated-test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testDataIngestSecretLocalAGWithMetrics = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedApiUrlDataIngestSecretLocalAgWithMetrics = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testDataIngestSecretLocalAGWithMetricsAndStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`

	testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`
	testEmptyFile = ``

	testNamespace1 = "test-namespace-one"
	testNamespace2 = "test-namespace-two"

	testNamespaceDynatrace = "dynatrace"
	testDynakubeName       = "dynakube"
)

func TestGenerateDataIngestSecret_ForDynakube(t *testing.T) {
	t.Run(`data-ingest endpoint secret created but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`data-ingest endpoint secret created and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}

		updateTestSecret(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedTokenDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`data-ingest endpoint secret created and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}

		updateTestDynakube(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})

	t.Run(`data-ingest endpoint secret created in all namespaces but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			testGenerateEndpointsSecret(t, instance, fakeClient)
		}
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
		}

		updateTestSecret(t, fakeClient)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedTokenDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testUpdatedTokenDataIngestSecretWithMetrics)
		}

		checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			newInstance := updatedTestDynakube()

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`data-ingest endpoint secret created (local AG) in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())
		{
			instance := buildTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAgWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAgWithMetrics)

			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
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

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetricsAndStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetricsAndStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`StatsD ingest URL is added/removed to endpoint properties when statsd-ingest capability is added/removed`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}

		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			newerInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newerInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
		{
			unchangedInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, unchangedInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testDataIngestSecretLocalAGWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`No ingestion is enabled (statsd capability is not enabled, disable-metadata-enrichment feature flag is set true)`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()
			instance.Annotations = map[string]string{
				dynatracev1beta1.AnnotationFeatureDisableMetadataEnrichment: "true",
			}

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName}, testEmptyFile)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName}, testEmptyFile)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: config.EnrichmentEndpointSecretName})
		}
	})
}

func testGenerateEndpointsSecret(t *testing.T, instance *dynatracev1beta1.DynaKube, fakeClient client.Client) {
	endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

	err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
	assert.NoError(t, err)
}

func TestRemoveEndpointSecrets(t *testing.T) {
	dk := buildTestDynakube()
	fakeClient := buildTestClientAfterGenerate(dk)

	endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, dk.Namespace)

	err := endpointSecretGenerator.RemoveEndpointSecrets(context.TODO(), dk)
	require.NoError(t, err)

	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: config.EnrichmentEndpointSecretName})
	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: config.EnrichmentEndpointSecretName})

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
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Data: map[string][]byte{
			"apiToken":        []byte(testApiToken),
			"paasToken":       []byte(testPaasToken),
			"dataIngestToken": []byte(testUpdatedDataIngestToken),
		},
	}

	err := fakeClient.Update(context.TODO(), updatedSecret)
	assert.NoError(t, err)
}

func updatedTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testUpdatedApiUrl,
		},
	}
}

func updatedTestDynakubeWithDataIngestCapability(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL: testUpdatedApiUrl,
		},
	}
}

func updateTestDynakube(t *testing.T, fakeClient client.Client) {
	var dk dynatracev1beta1.DynaKube
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: testDynakubeName, Namespace: testNamespaceDynatrace}, &dk)
	assert.NoError(t, err)

	dk.Spec.APIURL = testUpdatedApiUrl

	err = fakeClient.Update(context.TODO(), &dk)
	assert.NoError(t, err)
}

func buildTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
		},
	}
}

func buildTestDynakubeWithDataIngestCapability(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL: testApiUrl,
		},
	}
}

func buildTestClientBeforeGenerate(dk *dynatracev1beta1.DynaKube) client.Client {
	return fake.NewClient(dk,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespaceDynatrace,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace1,
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace2,
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(testApiToken),
				"paasToken":       []byte(testPaasToken),
				"dataIngestToken": []byte(testDataIngestToken),
			},
		})
}

func buildTestClientAfterGenerate(dk *dynatracev1beta1.DynaKube) client.Client {
	return fake.NewClient(dk,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespaceDynatrace,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace1,
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace2,
				Labels: map[string]string{
					dtwebhook.InjectionInstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testNamespace1,
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"doesn't": []byte("matter"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testNamespace2,
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"doesn't": []byte("matter"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(testApiToken),
				"paasToken":       []byte(testPaasToken),
				"dataIngestToken": []byte(testDataIngestToken),
			},
		})
}
