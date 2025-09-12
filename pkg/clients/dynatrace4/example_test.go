package dynatrace4

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace4/token"
	tokenMock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace4/token"
)

func TestDynatraceClient(t *testing.T) {
	// Replace these with actual values or load from environment/config
	baseURL := "https://BASE/api"
	apiToken := "DT_API_TOKEN_PLACEHOLDER"
	paasToken := "DT_PAAS_TOKEN_PLACEHOLDER"
	dataIngestToken := "DT_DATA_INGEST_TOKEN_PLACEHOLDER"

	client, err := NewClient(
		baseURL,
		WithAPIToken(apiToken),
		WithPaasToken(paasToken),
		WithDataIngestToken(dataIngestToken),
	)
	if err != nil {
		t.Fatalf("Failed to create Dynatrace client: %v", err)
	}

	t.Run("GetTokenScopes from tenant", func(t *testing.T) {
		scopes, err := client.Token().GetTokenScopes(t.Context(), apiToken)
		if err != nil {
			t.Errorf("GetTokenScopes error: %v", err)
		} else {
			t.Logf("Token scopes: %v", scopes)
		}
	})

	t.Run("GetTokenScopes but with a mock", func(t *testing.T) {
		tMock := tokenMock.NewClient(t)
		tMock.On("GetTokenScopes", t.Context(), apiToken).Return(token.TokenScopes{"sldkfj", "slkdfjlkj"}, nil)
		client := &Client{
			TokenClient: tMock,
		}

		scopes, err := client.Token().GetTokenScopes(t.Context(), apiToken)
		if err != nil {
			t.Errorf("GetTokenScopes error: %v", err)
		} else {
			t.Logf("Token scopes: %v", scopes)
		}
	})

	t.Run("GetRulesSettings", func(t *testing.T) {
		rules, err := client.Settings().GetRulesSettings(t.Context(), "asdfasf", "")
		if err != nil {
			t.Errorf("GetRulesSettings error: %v", err)
		} else {
			t.Logf("Rules settings: %v", rules)
		}
	})

	t.Run("GetK8sClusterMetadata", func(t *testing.T) {
		clusterSettings, err := client.Settings().GetK8sClusterMEDeleteThisMethod(t.Context(), "asdfasf")
		if err != nil {
			var httpErr *core.HTTPError
			if errors.As(err, &httpErr) {
				t.Logf("HTTP error: %v", httpErr)
				t.Logf("Status code: %d", httpErr.StatusCode)
				t.Logf("Status message: %s", httpErr.SingleError.Message)
			}
		} else {
			t.Logf("K8s cluster metadata: %v", clusterSettings)
		}
	})
}
