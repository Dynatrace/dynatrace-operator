package dynatrace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRulesSetting(t *testing.T) {
	ctx := context.Background()

	t.Run("get rules", func(t *testing.T) {
		mockParams := v2APIMockParams{
			entitiesAPI: entitiesMockParams{
				status:   http.StatusOK,
				expected: createMonitoredEntitiesForTesting(),
			},
			settingsAPI: settingsMockParams{
				status:     http.StatusOK,
				totalCount: 1,
				objectID:   "test",
			},
		}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(mockParams))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

		require.NoError(t, err)
		require.NotNil(t, dtc)

		rulesResponse, err := dtc.GetRulesSettings(ctx, "test-uuid", "test-entityID")
		require.NoError(t, err)
		assert.Equal(t, createRulesResponse(mockParams.settingsAPI.totalCount), rulesResponse)
	})

	t.Run("no kubesystem-uuid -> error", func(t *testing.T) {
		mockParams := v2APIMockParams{}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(mockParams))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

		require.NoError(t, err)
		require.NotNil(t, dtc)

		rulesResponse, err := dtc.GetRulesSettings(ctx, "", "test-entityID")
		require.Error(t, err)
		assert.Empty(t, rulesResponse)
	})

	t.Run("no monitored-entities, use environment scope -> return not-empty, no error", func(t *testing.T) {
		mockParams := v2APIMockParams{
			entitiesAPI: entitiesMockParams{
				status:   http.StatusOK,
				expected: []MonitoredEntity{},
			},
			settingsAPI: settingsMockParams{
				status:     http.StatusOK,
				totalCount: 1,
				objectID:   "test",
			},
		}

		dynatraceServer := httptest.NewServer(mockDynatraceServerV2Handler(mockParams))
		defer dynatraceServer.Close()

		skipCert := SkipCertificateValidation(true)
		dtc, err := NewClient(dynatraceServer.URL, apiToken, paasToken, skipCert)

		require.NoError(t, err)
		require.NotNil(t, dtc)

		rulesResponse, err := dtc.GetRulesSettings(ctx, "test-uuid", "")
		require.NoError(t, err)
		assert.NotEmpty(t, rulesResponse)
	})
}

func mockGetRulesSettingsAPI(writer http.ResponseWriter, totalCount int) {
	rawResponse, err := json.Marshal(createRulesResponse(totalCount))
	if err != nil {
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(rawResponse)
}

func createRulesResponse(totalCount int) GetRulesSettingsResponse {
	rules := []dynakube.EnrichmentRule{
		{
			Source: "rule-1",
		},
		{
			Source: "rule-2",
		},
	}
	rulesGetResponse := GetRulesSettingsResponse{
		TotalCount: totalCount,
		Items: []RuleItem{
			{
				Value: RulesResponseValue{
					Rules: rules,
				},
			},
		},
	}

	return rulesGetResponse
}
