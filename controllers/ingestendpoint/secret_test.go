package ingestendpoint

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken              = "test-paas-token"
	testAPIToken               = "test-api-token"
	testDataIngestToken        = "test-data-ingest-token"
	testUpdatedDataIngestToken = "updated-test-data-ingest-token"

	testApiUrl        = "https://test/api"
	testUpdatedApiUrl = "https://updated-test/api"

	testDataIngestSecret = `DT_METRICS_INGEST_URL=https://test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testUpdatedTokenDataIngestSecret = `DT_METRICS_INGEST_URL=https://test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=updated-test-data-ingest-token
`
	testUpdatedApiUrlDataIngestSecret = `DT_METRICS_INGEST_URL=https://updated-test/api/v2/metrics/ingest
DT_METRICS_INGEST_API_TOKEN=test-data-ingest-token
`
	testNamespace1 = "test-namespace-one"
	testNamespace2 = "test-namespace-two"

	testNamespaceDynatrace = "dynatrace"
	testDynakubeName       = "dynakube"
)

var log = logger.NewDTLogger()

func TestGenerateDataIngestSecret_ForDynakube(t *testing.T) {
	t.Run(`data-ingest endpoint secret created but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		upd, err = endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, false, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)
	})
	t.Run(`data-ingest endpoint secret created and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		updateTestSecret(t, fakeClient)

		upd, err = endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testUpdatedTokenDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)
	})
	t.Run(`data-ingest endpoint secret created and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		updateTestDynakube(t, fakeClient)

		upd, err = endpointSecretGenerator.GenerateForNamespace(context.TODO(), testDynakubeName, testNamespace1)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)
	})

	t.Run(`data-ingest endpoint secret created in all namespaces but not updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testDataIngestSecret)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2, testDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)

		upd, err = endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
		assert.NoError(t, err)
		assert.Equal(t, false, upd)
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and token updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		updateTestSecret(t, fakeClient)

		upd, err = endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testUpdatedTokenDataIngestSecret)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2, testUpdatedTokenDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)
	})
	t.Run(`data-ingest endpoint secret created in all namespaces and apiUrl updated`, func(t *testing.T) {
		instance := buildTestDynakube()
		fakeClient := buildTestClient(instance)

		endpointSecretGenerator := NewEndpointGenerator(fakeClient, fakeClient, testNamespaceDynatrace, log)

		upd, err := endpointSecretGenerator.GenerateForDynakube(context.TODO(), instance)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		newInstance := updatedTestDynakube()

		upd, err = endpointSecretGenerator.GenerateForDynakube(context.TODO(), newInstance)
		assert.NoError(t, err)
		assert.Equal(t, true, upd)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace1, testUpdatedApiUrlDataIngestSecret)

		checkTestSecretExists(t, fakeClient, webhook.SecretEndpointName, testNamespace2, testUpdatedApiUrlDataIngestSecret)

		checkTestSecretNotExists(t, fakeClient, webhook.SecretEndpointName, testNamespaceDynatrace)
	})
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

func buildTestClient(dk *dynatracev1beta1.DynaKube) client.Client {
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
				Name:      testDynakubeName,
				Namespace: testNamespaceDynatrace,
			},
			Data: map[string][]byte{
				"apiToken":        []byte(testAPIToken),
				"paasToken":       []byte(testPaasToken),
				"dataIngestToken": []byte(testDataIngestToken),
			},
		})
}
