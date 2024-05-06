package ingestendpoint

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
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

	testTenant        = "tenant"
	testApiUrl        = "https://tenant.test/api"
	testUpdatedApiUrl = "https://tenant.updated-test/api"

	testMetadataEnrichmentSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedTokenMetadataEnrichmentSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=updated-test-data-ingest-token
`
	testUpdatedApiUrlMetadataEnrichmentSecretWithMetrics = `DT_METRICS_INGEST_URL=https://tenant.updated-test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testMetadataEnrichmentSecretLocalAGWithMetrics = `DT_METRICS_INGEST_URL=http://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedApiUrlMetadataEnrichmentSecretLocalAgWithMetrics = `DT_METRICS_INGEST_URL=http://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testEmptyFile = ``

	testNamespace1 = "test-namespace-one"
	testNamespace2 = "test-namespace-two"

	testNamespaceDynatrace = "dynatrace"
	testDynakubeName       = "dynakube"
)

func TestGenerateMetadataEnrichmentSecret_ForDynakube(t *testing.T) {
	t.Run(`metadata-enrichment endpoint secret created but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`metadata-enrichment endpoint secret created and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}

		updateTestSecret(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testUpdatedTokenMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`metadata-enrichment endpoint secret created and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}

		updateTestDynakube(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			require.NoError(t, err)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testUpdatedApiUrlMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})

	t.Run(`metadata-enrichment endpoint secret created in all namespaces but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
		{
			testGenerateEndpointsSecret(t, instance, fakeClient)
		}
	})
	t.Run(`metadata-enrichment endpoint secret created in all namespaces and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
		}

		updateTestSecret(t, fakeClient)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testUpdatedTokenMetadataEnrichmentSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testUpdatedTokenMetadataEnrichmentSecretWithMetrics)
		}

		checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
	})
	t.Run(`metadata-enrichment endpoint secret created in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
		{
			newInstance := updatedTestDynakube()

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testUpdatedApiUrlMetadataEnrichmentSecretWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testUpdatedApiUrlMetadataEnrichmentSecretWithMetrics)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`metadata-enrichment endpoint secret created (local AG) in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())
		{
			instance := buildTestDynakubeWithMetricsIngestCapability([]dynatracev1beta2.CapabilityDisplayName{
				dynatracev1beta2.CapabilityDisplayName(dynatracev1beta2.KubeMonCapability.ShortName),
				dynatracev1beta2.CapabilityDisplayName(dynatracev1beta2.MetricsIngestCapability.ShortName),
			})
			addFakeTenantUUID(instance)

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretLocalAGWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testMetadataEnrichmentSecretLocalAGWithMetrics)
		}
		{
			newInstance := updatedTestDynakubeWithMetricsIngestCapability([]dynatracev1beta2.CapabilityDisplayName{
				dynatracev1beta2.CapabilityDisplayName(dynatracev1beta2.KubeMonCapability.ShortName),
				dynatracev1beta2.CapabilityDisplayName(dynatracev1beta2.MetricsIngestCapability.ShortName),
			})
			addFakeTenantUUID(newInstance)

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testUpdatedApiUrlMetadataEnrichmentSecretLocalAgWithMetrics)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testUpdatedApiUrlMetadataEnrichmentSecretLocalAgWithMetrics)

			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})
	t.Run(`No ingestion is enabled (metadaEnrichment.enabled is set to false)`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()
			instance.Spec.MetadataEnrichment.Enabled = false

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName}, testEmptyFile)
			checkTestSecretContains(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName}, testEmptyFile)
			checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespaceDynatrace, Name: consts.EnrichmentEndpointSecretName})
		}
	})
}

func addFakeTenantUUID(dynakube *dynatracev1beta2.DynaKube) *dynatracev1beta2.DynaKube {
	dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = testTenant

	return dynakube
}

func testGenerateEndpointsSecret(t *testing.T, instance *dynatracev1beta2.DynaKube, fakeClient client.Client) {
	endpointSecretGenerator := NewSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

	err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
	require.NoError(t, err)
}

func TestRemoveEndpointSecrets(t *testing.T) {
	dk := buildTestDynakube()
	fakeClient := buildTestClientAfterGenerate(dk)

	namespaces, err := mapper.GetNamespacesForDynakube(context.Background(), fakeClient, dk.Name)
	require.NoError(t, err)

	endpointSecretGenerator := NewSecretGenerator(fakeClient, fakeClient, dk.Namespace)

	err = endpointSecretGenerator.RemoveEndpointSecrets(context.TODO(), namespaces)
	require.NoError(t, err)

	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace1, Name: consts.EnrichmentEndpointSecretName})
	checkTestSecretDoesntExist(t, fakeClient, types.NamespacedName{Namespace: testNamespace2, Name: consts.EnrichmentEndpointSecretName})
}

func checkTestSecretContains(t *testing.T, fakeClient client.Client, secretName types.NamespacedName, data string) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), secretName, &testSecret)
	require.NoError(t, err)
	assert.NotNil(t, testSecret.Data)
	assert.NotEmpty(t, testSecret.Data)
	assert.Contains(t, testSecret.Data, "endpoint.properties")
	assert.Equal(t, data, string(testSecret.Data["endpoint.properties"]))
}

func checkTestSecretDoesntExist(t *testing.T, fakeClient client.Client, secretName types.NamespacedName) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), secretName, &testSecret)
	require.Error(t, err)
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
	require.NoError(t, err)
}

func updatedTestDynakube() *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL:             testUpdatedApiUrl,
			MetadataEnrichment: dynatracev1beta2.MetadataEnrichment{Enabled: true},
		},
	}
}

func updatedTestDynakubeWithMetricsIngestCapability(capabilities []dynatracev1beta2.CapabilityDisplayName) *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			ActiveGate: dynatracev1beta2.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL:             testUpdatedApiUrl,
			MetadataEnrichment: dynatracev1beta2.MetadataEnrichment{Enabled: true},
		},
	}
}

func updateTestDynakube(t *testing.T, fakeClient client.Client) {
	var dk dynatracev1beta2.DynaKube
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: testDynakubeName, Namespace: testNamespaceDynatrace}, &dk)
	require.NoError(t, err)

	dk.Spec.APIURL = testUpdatedApiUrl

	err = fakeClient.Update(context.TODO(), &dk)
	require.NoError(t, err)
}

func buildTestDynakube() *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL:             testApiUrl,
			MetadataEnrichment: dynatracev1beta2.MetadataEnrichment{Enabled: true},
		},
	}
}

func buildTestDynakubeWithMetricsIngestCapability(capabilities []dynatracev1beta2.CapabilityDisplayName) *dynatracev1beta2.DynaKube {
	dynakube := &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			ActiveGate: dynatracev1beta2.ActiveGateSpec{
				Capabilities: capabilities,
			},
			APIURL: testApiUrl,
			MetadataEnrichment: dynatracev1beta2.MetadataEnrichment{
				Enabled: true,
			},
		},
	}

	return addFakeTenantUUID(dynakube)
}

func buildTestClientBeforeGenerate(dk *dynatracev1beta2.DynaKube) client.Client {
	return fake.NewClientWithIndex(dk,
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

func buildTestClientAfterGenerate(dk *dynatracev1beta2.DynaKube) client.Client {
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
