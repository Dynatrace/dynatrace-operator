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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken              = "test-paas-token"
	testAPIToken               = "test-api-token"
	testDataIngestToken        = "test-data-ingest-token"
	testUpdatedDataIngestToken = "updated-test-data-ingest-token"

	testApiUrl        = "https://tenant.test/api"
	testUpdatedApiUrl = "https://tenant.updated-test/api"

	testDataIngestSecret = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedTokenDataIngestSecret = `DT_METRICS_INGEST_URL=https://tenant.test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=updated-test-data-ingest-token
`
	testUpdatedApiUrlDataIngestSecret = `DT_METRICS_INGEST_URL=https://tenant.updated-test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testDataIngestSecretLocalAG = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedApiUrlDataIngestSecretLocalAG = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`

	testDataIngestSecretLocalAGWithStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`

	testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd = `DT_METRICS_INGEST_URL=https://dynakube-activegate.dynatrace/e/tenant/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
DT_STATSD_INGEST_URL=dynakube-activegate.dynatrace:18125
`

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

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
	})
	t.Run(`data-ingest endpoint secret created and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}

		updateTestSecret(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedTokenDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
	})
	t.Run(`data-ingest endpoint secret created and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)
		endpointSecretGenerator := NewEndpointSecretGenerator(fakeClient, fakeClient, testNamespaceDynatrace)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}

		updateTestDynakube(t, fakeClient)

		{
			err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
			assert.NoError(t, err)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
	})

	t.Run(`data-ingest endpoint secret created in all namespaces but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClientBeforeGenerate(instance)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecret)

			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
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

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecret)
		}

		updateTestSecret(t, fakeClient)

		{
			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedTokenDataIngestSecret)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testUpdatedTokenDataIngestSecret)
		}

		checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and apiUrl updated`, func(t *testing.T) {
		fakeClient := buildTestClientBeforeGenerate(buildTestDynakube())

		{
			instance := buildTestDynakube()

			testGenerateEndpointsSecret(t, instance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecret)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
		{
			newInstance := updatedTestDynakube()

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecret)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testUpdatedApiUrlDataIngestSecret)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
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

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecretLocalAG)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecretLocalAG)
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecretLocalAG)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testUpdatedApiUrlDataIngestSecretLocalAG)

			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
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

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecretLocalAGWithStatsd)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecretLocalAGWithStatsd)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
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

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecretLocalAG)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecretLocalAG)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}

		{
			newInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.StatsdIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testUpdatedApiUrlDataIngestSecretLocalAGWithStatsd)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
		{
			newerInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, newerInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecretLocalAG)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecretLocalAG)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
		}
		{
			unchangedInstance := updatedTestDynakubeWithDataIngestCapability([]dynatracev1beta1.CapabilityDisplayName{
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.KubeMonCapability.ShortName),
				dynatracev1beta1.CapabilityDisplayName(dynatracev1beta1.MetricsIngestCapability.ShortName),
			})

			testGenerateEndpointsSecret(t, unchangedInstance, fakeClient)

			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace1, testDataIngestSecretLocalAG)
			checkTestSecretExists(t, fakeClient, SecretEndpointName, testNamespace2, testDataIngestSecretLocalAG)
			checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespaceDynatrace)
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

	checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace1)
	checkTestSecretNotExists(t, fakeClient, SecretEndpointName, testNamespace2)

}

func checkTestSecretExists(t *testing.T, fakeClient client.Client, secretName string, namespace string, data string) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: namespace}, &testSecret)
	assert.NoError(t, err)
	assert.NotNil(t, testSecret.Data)
	assert.NotEmpty(t, testSecret.Data)
	assert.Contains(t, testSecret.Data, "endpoint.properties")
	assert.Equal(t, data, string(testSecret.Data["endpoint.properties"]))
}

func checkTestSecretNotExists(t *testing.T, fakeClient client.Client, secretName string, namespace string) {
	var testSecret corev1.Secret
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: namespace}, &testSecret)
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
			"apiToken":        []byte(testAPIToken),
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
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace2,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.Tokens(),
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(testAPIToken),
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
					mapper.InstanceLabel: dk.Name,
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace2,
				Labels: map[string]string{
					mapper.InstanceLabel: dk.Name,
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
				"apiToken":        []byte(testAPIToken),
				"paasToken":       []byte(testPaasToken),
				"dataIngestToken": []byte(testDataIngestToken),
			},
		})
}
