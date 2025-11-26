package token

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testAPIToken        = "test-api-token"
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
		dk := dynakube.DynaKube{}
		reader := NewReader(clt, &dk)

		_, err := reader.readTokens(context.Background())

		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("tokens are found if secret exists", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube",
				Namespace: "dynatrace",
			},
		}
		testSecret, err := k8ssecret.Build(&dk, "dynakube", map[string][]byte{
			dtclient.APIToken:        []byte(testAPIToken),
			dtclient.PaasToken:       []byte(testPaasToken),
			dtclient.DataIngestToken: []byte(testDataIngestToken),
			testIrrelevantTokenKey:   []byte(testIrrelevantToken),
		})
		require.NoError(t, err)

		clt := fake.NewClient(testSecret, &dk)

		reader := NewReader(clt, &dk)

		tokens, err := reader.readTokens(context.Background())

		require.NoError(t, err)
		assert.Len(t, tokens, 4)
		assert.Contains(t, tokens, dtclient.APIToken)
		assert.Contains(t, tokens, dtclient.PaasToken)
		assert.Contains(t, tokens, dtclient.DataIngestToken)
		assert.Contains(t, tokens, testIrrelevantTokenKey)
		assert.Equal(t, testAPIToken, tokens[dtclient.APIToken].Value)
		assert.Equal(t, testPaasToken, tokens[dtclient.PaasToken].Value)
		assert.Equal(t, testDataIngestToken, tokens[dtclient.DataIngestToken].Value)
		assert.Equal(t, testIrrelevantToken, tokens[testIrrelevantTokenKey].Value)
	})
}

func testVerifyTokens(t *testing.T) {
	t.Run("error if api token is missing", func(t *testing.T) {
		reader := NewReader(nil, &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: dynatraceNamespace,
		}})

		err := reader.verifyAPITokenExists(map[string]*Token{
			testIrrelevantTokenKey: {
				Value: testIrrelevantToken,
			},
		})

		require.EqualError(t, err, "the API token is missing from the token secret 'dynatrace:dynakube'")
	})
	t.Run("no error if api token exists", func(t *testing.T) {
		reader := NewReader(nil, nil)

		err := reader.verifyAPITokenExists(map[string]*Token{
			testIrrelevantTokenKey: {
				Value: testIrrelevantToken,
			},
			dtclient.APIToken: {
				Value: testAPIToken,
			},
		})

		require.NoError(t, err)
	})
}
