package token

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testApiToken        = "test-api-token"
	testPaasToken       = "test-paas-token"
	testDataIngestToken = "test-data-ingest-token"
	testIrrelevantToken = "test-irrelevant-token"

	testIrrelevantTokenKey = "irrelevant-token"

	dynakubeName       = "dynakube"
	dynatraceNamespace = "dynatrace"
)

func TestReader(t *testing.T) {
	t.Run("read tokens", testReadTokens)
	t.Run("verify tokens", testVerifyTokens)
}

func testReadTokens(t *testing.T) {
	t.Run("error when tokens are not found", func(t *testing.T) {
		clt := fake.NewClient()
		dynakube := dynatracev1beta1.DynaKube{}
		reader := NewReader(clt, &dynakube)

		_, err := reader.readTokens(context.Background())

		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("tokens are found if secret exists", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
		}
		secret, err := kubeobjects.CreateSecret(scheme.Scheme, &dynakube, kubeobjects.NewSecretNameModifier("dynakube"), kubeobjects.NewSecretNamespaceModifier("dynatrace"), kubeobjects.NewSecretDataModifier(map[string][]byte{
			dtclient.DynatraceApiToken:        []byte(testApiToken),
			dtclient.DynatracePaasToken:       []byte(testPaasToken),
			dtclient.DynatraceDataIngestToken: []byte(testDataIngestToken),
			testIrrelevantTokenKey:            []byte(testIrrelevantToken),
		}))
		assert.NoError(t, err)
		clt := fake.NewClient(secret, &dynakube)

		reader := NewReader(clt, &dynakube)

		tokens, err := reader.readTokens(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, 4, len(tokens))
		assert.Contains(t, tokens, dtclient.DynatraceApiToken)
		assert.Contains(t, tokens, dtclient.DynatracePaasToken)
		assert.Contains(t, tokens, dtclient.DynatraceDataIngestToken)
		assert.Contains(t, tokens, testIrrelevantTokenKey)
		assert.Equal(t, tokens[dtclient.DynatraceApiToken].Value, testApiToken)
		assert.Equal(t, tokens[dtclient.DynatracePaasToken].Value, testPaasToken)
		assert.Equal(t, tokens[dtclient.DynatraceDataIngestToken].Value, testDataIngestToken)
		assert.Equal(t, tokens[testIrrelevantTokenKey].Value, testIrrelevantToken)
	})
}

func testVerifyTokens(t *testing.T) {
	t.Run("error if api token is missing", func(t *testing.T) {
		reader := NewReader(nil, &dynatracev1beta1.DynaKube{ObjectMeta: v1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: dynatraceNamespace,
		}})

		err := reader.verifyApiTokenExists(map[string]Token{
			testIrrelevantTokenKey: {
				Value: testIrrelevantToken,
			},
		})

		assert.EqualError(t, err, "the API token is missing from the token secret 'dynatrace:dynakube'")
	})
	t.Run("no error if api token exists", func(t *testing.T) {
		reader := NewReader(nil, nil)

		err := reader.verifyApiTokenExists(map[string]Token{
			testIrrelevantTokenKey: {
				Value: testIrrelevantToken,
			},
			dtclient.DynatraceApiToken: {
				Value: testApiToken,
			},
		})

		assert.NoError(t, err)
	})
}
