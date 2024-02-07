package token

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("tokens are found if secret exists", func(t *testing.T) {
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
		}
		secret, err := secret.Create(scheme.Scheme, &dynakube, secret.NewNameModifier("dynakube"), secret.NewNamespaceModifier("dynatrace"), secret.NewDataModifier(map[string][]byte{
			dtclient.DynatraceApiToken:        []byte(testApiToken),
			dtclient.DynatracePaasToken:       []byte(testPaasToken),
			dtclient.DynatraceDataIngestToken: []byte(testDataIngestToken),
			testIrrelevantTokenKey:            []byte(testIrrelevantToken),
		}))
		require.NoError(t, err)

		clt := fake.NewClient(secret, &dynakube)

		reader := NewReader(clt, &dynakube)

		tokens, err := reader.readTokens(context.Background())

		require.NoError(t, err)
		assert.Len(t, tokens, 4)
		assert.Contains(t, tokens, dtclient.DynatraceApiToken)
		assert.Contains(t, tokens, dtclient.DynatracePaasToken)
		assert.Contains(t, tokens, dtclient.DynatraceDataIngestToken)
		assert.Contains(t, tokens, testIrrelevantTokenKey)
		assert.Equal(t, testApiToken, tokens[dtclient.DynatraceApiToken].Value)
		assert.Equal(t, testPaasToken, tokens[dtclient.DynatracePaasToken].Value)
		assert.Equal(t, testDataIngestToken, tokens[dtclient.DynatraceDataIngestToken].Value)
		assert.Equal(t, testIrrelevantToken, tokens[testIrrelevantTokenKey].Value)
	})
}

func testVerifyTokens(t *testing.T) {
	t.Run("error if api token is missing", func(t *testing.T) {
		reader := NewReader(nil, &dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: dynatraceNamespace,
		}})

		err := reader.verifyApiTokenExists(map[string]Token{
			testIrrelevantTokenKey: {
				Value: testIrrelevantToken,
			},
		})

		require.EqualError(t, err, "the API token is missing from the token secret 'dynatrace:dynakube'")
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

		require.NoError(t, err)
	})
}
